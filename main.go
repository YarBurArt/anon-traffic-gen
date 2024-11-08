package main

import (
    "fmt"
    "path/filepath"
    "math/rand"
    "io/ioutil"
    "net/http"
    "regexp"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v2"
    "github.com/spf13/viper"
    "github.com/anacrolix/torrent"
)

type Config struct {
    // mapstructure decode json to Go map[string]interface{} or same 
    URLs       []string `mapstructure:"urls"`
    UserAgents []string `mapstructure:"user_agents"`
    RateLimit    int      `mapstructure:"rate_limit"`
    Timeout      int      `mapstructure:"timeout"`
    MaxRetries   int      `mapstructure:"max_retries"`
    WebSocketTimeout int  `mapstructure:"websocket_timeout"`
    TorrentLink  string   `mapstructure:"torrent_link"`
}

var rootCmd = &cobra.Command{ 
    // cobra instance for easy use app by CLI, it is used from the main entry point below  
    Use:   "my-app",
    Short: "A simple application to generate HTTP traffic",
    Run: func(cmd *cobra.Command, args []string) { 
        // entry point for CLI part of the program
        // cmd - current instance, args for todo --config, -h ...  
        config := loadConfig()

        fmt.Println("Starting HTTP traffic generation...")
        stop := make(chan struct{})

        go sendRequests(config, stop)
        // more effective on tiny but popular torrents, where a lot of IP for content distribution
        go downloadFileTorrent(config.TorrentLink, config.MaxRetries) // some time for ip spoof 

        handleSignals(stop)
        fmt.Println("Press Ctrl+C to stop")
        <-stop // wait to stop signal
    },
}

func handleSignals(stop chan struct{}) { 
    // handle make(chan struct{}) if it is like ctrl+c 
    // or SIGINT, SIGTERM, SIGTSTP from unix process termination
    signalChan := make(chan os.Signal, 1) // correct handle termination signals
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)
    
    go func() { <-signalChan;close(stop) }()
    // to original dir before removing the tmp folder
    defer func() {
        os.Chdir(".."); // remove the folder on exit 
        os.RemoveAll(filepath.Join(".", "tmp_data_t"));
    }()
}

func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().String("config", "config.yaml", "config file (default is config.yaml)")
    viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
}

func initConfig() {
    viper.SetConfigType("yaml")
    viper.AutomaticEnv()
    configFile := viper.GetString("config")
    if configFile != "" {
        viper.SetConfigFile(configFile)
        if err := viper.ReadInConfig(); err != nil {
            fmt.Println("Error reading config file:", err)
        }
    }
}

func loadConfig() Config {
    var config Config
    err := viper.Unmarshal(&config)
    if err != nil {
        fmt.Println("Unable to decode into struct, ", err)
    }
    return config
}

func downloadFileTorrent(magnetURI string, maxAttempts int) {
    // temp downloads torrent via p2p network for IP connections only
    downloadDir := filepath.Join(".", "tmp_data_t")
    os.MkdirAll(downloadDir, os.ModePerm)
    
    if err := os.Chdir(downloadDir); err != nil { // cd ./tmp_data_t
        fmt.Println("Error changing directory:", err)
        return
    }

    for attempts := 0; attempts < maxAttempts; attempts++ {
        // downloading a small torrent several times for IP difference
        client, _ := torrent.NewClient(nil)
        t, _ := client.AddMagnet(magnetURI)
        
        // Wait for the download to complete
        <-t.GotInfo()
        t.DownloadAll()
        client.WaitAll()
        fi := t.Files()[0]
        fileName := filepath.Base(fi.Path())
        filePath := filepath.Join(downloadDir, fileName)

        os.Remove(filePath) // no files needed, just the IP connections
        if attempts < maxAttempts-1 {
            time.Sleep(5 * time.Second)
        }
        // Cleanup the torrent client
        client.Close()
    }
}

func sendRequests(config Config, stop chan struct{}) {
    // wrapper for correct mixing of http and WS traffic 
    for {
        select {
        case <-stop:
            return
        default:
            // Send HTTP requests
            sendHTTPRequests(config)

            // Randomly send WebSocket traffic
            if rand.Intn(5) == 0 { // 20% chance of sending WebSocket traffic
                generateWebSocketTraffic(config)
            }

            // Add delay for rate limiting
            time.Sleep(time.Second * time.Duration(config.RateLimit))
        }
    }
}

func sendHTTPRequests(config Config) { 
    // mixing urls and user agents, sending requests, new urls to the config 
    for _, url := range config.URLs {
        for _, ua := range config.UserAgents {
            req, err := http.NewRequest(http.MethodGet, url, nil)
            if err != nil {
                fmt.Println("Error creating request:", err)
                continue
            }
            req.Header.Set("User-Agent", ua)

            // send request and measure time
            start := time.Now()
            client := &http.Client{}
            resp, err := client.Do(req)
            elapsed := time.Since(start)

            if err != nil {
                fmt.Println("Error sending request:", err)
                continue
            }
            defer resp.Body.Close()

            // output current request status 
            fmt.Printf("Request: %s %s \n", http.MethodGet, url);
            fmt.Printf("Response: Status Code: %d, Elapsed Time: %s\n", resp.StatusCode, elapsed)
            
            // TODO: refactoring here 
            // Parse response body for additional URLs
            body, err := ioutil.ReadAll(resp.Body)
            if err != nil {
                fmt.Println("Error reading response body:", err)
                continue
            }

            // new urls to  config
            links := findURLs(string(body))
            for _, link := range links {
                if !contains(config.URLs, link) {
                    config.URLs = append(config.URLs, link)
                    fmt.Printf("Added new URL to config: %s\n", link)
                }
            }
        }
    }
    saveConfig(config, "config.yaml") // update urls
}

func findURLs(text string) []string {
    // regex to find all URLs in response, FIXME: some html after url
    urlRegex := regexp.MustCompile(`(https?://\S+)`)
    return urlRegex.FindAllString(text, -1)
}

func contains(slice []string, value string) bool {
    for _, v := range slice {
        if v == value {
            return true
        }
    }
    return false
}

func saveConfig(config Config, filename string) {
    // update config to YAML file
    data, err := yaml.Marshal(config)
    if err != nil {
        fmt.Println("Error marshaling config:", err)
        return
    }
    // 0644 is Unix permissions
    err = ioutil.WriteFile(filename, data, 0644)
    if err != nil {
        fmt.Println("Error writing config file:", err)
        return
    }
}
func generateWebSocketTraffic(config Config) {
    // gently and somewhat interestingly generates some of the WS traffic

    dialer := websocket.Dialer{
        HandshakeTimeout: time.Duration(config.WebSocketTimeout) * time.Second, 
    }

    for _, url := range config.URLs {
        for _, ua := range config.UserAgents {
            // Establish WebSocket connection
            headers := http.Header{}
            headers.Set("User-Agent", ua)
            conn, _, err := dialer.Dial(url, headers)
            if err != nil {
                fmt.Println("Error connecting to WebSocket:", err)
                continue
            }
            defer conn.Close()

            // Send message and immediately close the connection
            message := []byte(fmt.Sprintf("WS message: %d", rand.Intn(1000)))
            err = conn.WriteMessage(websocket.TextMessage, message)
            if err != nil {
                fmt.Println("Error sending WebSocket message:", err)
            }
        }
    }
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
