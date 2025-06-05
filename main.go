package main

import (
    "fmt"
    "log"
    "context"
    "path/filepath"
    "math/rand"
    "io/ioutil"
    "net/http"
    "regexp"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v2"
    "github.com/spf13/viper"
    "github.com/anacrolix/torrent"
)

// mapstructure decode json to Go map[string]interface{} or same 
type Config struct {
    URLs       []string `mapstructure:"urls"`
    UserAgents []string `mapstructure:"user_agents"`
    RateLimit    int      `mapstructure:"rate_limit"`
    Timeout      int      `mapstructure:"timeout"`
    maxRetries   int      `mapstructure:"max_retries"`
    WebSocketTimeout int  `mapstructure:"websocket_timeout"`
    torrentLink  string   `mapstructure:"torrent_link"`
}

var rootCmd = &cobra.Command{ 
    // cobra instance for easy use app by CLI, it is used from the main entry point below  
    Use:   "main",
    Short: "A simple application to generate HTTP traffic",
    Run: func(cmd *cobra.Command, args []string) { 
        // entry point for CLI part of the program
        // cmd - current instance, args for todo --config, -h ...  
        config := loadConfig()

        fmt.Println("Starting HTTP traffic generation...")
        stop := make(chan struct{})

        ctx, cancel := context.WithCancel(context.Background())
        defer cancel() 
        go handleSignals(stop, cancel)
        var wg sync.WaitGroup
        wg.Add(2)

        go func() {
            defer wg.Done()
            sendRequests(ctx, config, stop)
        }()

        go func() {
            defer wg.Done()
            // more effective on tiny but popular torrents, where a lot of IP for content distribution 
            downloadFileTorrent(ctx, config.torrentLink, config.maxRetries)
        }()

        fmt.Println("Press Ctrl+C to stop")
        wg.Wait()
    },
}

func handleSignals(stop chan struct{}, cancel context.CancelFunc) { 
    // handle make(chan struct{}) if it is like ctrl+c 
    // or SIGINT, SIGTERM, SIGTSTP from unix process termination
    signalChan := make(chan os.Signal, 1) // correct handle termination signals
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

    go func() {
        <-signalChan
        log.Println("Received stop signal, cancelling context...")
        cancel() // reset context
        close(stop)
    }()    // to original dir before removing the tmp folder
    defer func() { // remove tmp with torrents
        if err := os.Chdir(".."); err != nil {
            log.Printf("Error changing directory to parent: %v", err)
            if err := os.RemoveAll(filepath.Join(".", "tmp_data_t")); err != nil {
                log.Printf("Error removing tmp_data_t directory: %v", err)
            }
        }
    }()
}

func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().String(
        "config", "config.yaml", "config file (default is config.yaml)")
    rootCmd.PersistentFlags().Bool(
        "debug", false, "enable debug log")
    viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
    viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func initConfig() {
    viper.SetConfigType("yaml")
    viper.AutomaticEnv()
    configFile := viper.GetString("config")
    if configFile != "" {
        viper.SetConfigFile(configFile)
        if err := viper.ReadInConfig(); err != nil {
            log.Println("Error reading config file:", err)
        }
    }
}

func loadConfig() Config {
    var config Config
    err := viper.Unmarshal(&config)
    if err != nil {
        log.Printf("Error validating config: %v", err)
    }
    return config
}

func downloadFileTorrent(ctx context.Context, magnetURI string, maxAttempts int) {
    // temp downloads torrent via p2p network for IP connections only
    downloadDir := filepath.Join(".", "tmp_data_t")
    if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil { // mkdir ./tmp_data_t
        log.Printf("Error creating directory %s: %v", downloadDir, err)
        if err := os.Chdir(downloadDir); err != nil { // cd ./tmp_data_t
            log.Printf("Error changing directory to %s: %v", downloadDir, err)
            return
        }
        return
    }
    select {
    case <- ctx.Done():
        log.Println("Stop torrent ...")
        return
    default:
        log.Println("starting torrent")
        if viper.GetBool("debug") {
            log.Printf("URI: %s", magnetURI)
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
}

func sendRequests(ctx context.Context, config Config, stop chan struct{}) {
    // wrapper for correct mixing of http and WS traffic 
    for {
        select {
        case <-ctx.Done():
            log.Println("Stop http requests...")
            return
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
                log.Printf("Error in sendHTTPRequests: %v", err)
                continue
            }
            if viper.GetBool("debug") {
                log.Println("url x user agent combination: ", url, ua)
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
                log.Println("Error sending WebSocket message:", err)
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
