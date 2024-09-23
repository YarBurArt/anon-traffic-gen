package main

import (
    "fmt"
    "path/filepath"
    "math/rand"
    "io/ioutil"
    "net/http"
    "regexp"
    "os"
    "time"

    "github.com/gorilla/websocket"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v2"
    "github.com/spf13/viper"
    "github.com/anacrolix/torrent"
)

type Config struct {
    URLs       []string `mapstructure:"urls"`
    UserAgents []string `mapstructure:"user_agents"`
    RateLimit  int      `mapstructure:"rate_limit"`
}

var rootCmd = &cobra.Command{
    Use:   "my-app",
    Short: "A simple application to generate HTTP traffic",
    Run: func(cmd *cobra.Command, args []string) {
        config := loadConfig()

        fmt.Println("Starting HTTP traffic generation...")

        stop := make(chan struct{})

        go sendRequests(config, stop)
        go downloadFile("magnet:?xt=urn:btih:723924A57607F3C926A644271E5B40BDF0B36F29&dn=IObit+Driver+Booster+Pro+12.0.0.308+Cracked+-+%5BCZSofts%5D&tr=http%3A%2F%2Fbt02.nnm-club.cc%3A2710%2F00b89bb6cf2713fa8a7b67da0f5dc8ee%2Fannounce&tr=http%3A%2F%2Fbt02.nnm-club.info%3A2710%2F00b89bb6cf2713fa8a7b67da0f5dc8ee%2Fannounce&tr=http%3A%2F%2Fretracker.local%2Fannounce.php%3Fsize%3D31539444%26comment%3Dhttp%253A%252F%252Fnnmclub.to%252Fforum%252Fviewtopic.php%253Fp%253D12483744%26name%3DIObit%2BDriver%2BBooster%2BPro%2B12.0.0.308%2BRePack%2B%2528%2526amp%253B%2BPortable%2529%2Bby%2BTryRooM%2B%255BMulti%252FRu%255D&tr=http%3A%2F%2Fbt02.ipv6.nnm-club.cc%3A2710%2F00b89bb6cf2713fa8a7b67da0f5dc8ee%2Fannounce&tr=http%3A%2F%2Fbt02.ipv6.nnm-club.info%3A2710%2F00b89bb6cf2713fa8a7b67da0f5dc8ee%2Fannounce&tr=http%3A%2F%2F%5B2a01%3Ad0%3Aa580%3A1%3A%3A2%5D%3A2710%2F00b89bb6cf2713fa8a7b67da0f5dc8ee%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=http%3A%2F%2Ftracker.openbittorrent.com%3A80%2Fannounce&tr=udp%3A%2F%2Fopentracker.i2p.rocks%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.internetwarriors.net%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969%2Fannounce&tr=udp%3A%2F%2Fcoppersurfer.tk%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.zer0day.to%3A1337%2Fannounce",10) // some time for ip spoof 
        fmt.Println("Press Enter to stop")
        fmt.Scanln()

        close(stop)
    },
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

func downloadFile(magnetURI string, maxAttempts int) {
    for attempts := 0; attempts < maxAttempts; attempts++ {
        client, _ := torrent.NewClient(nil)
        t, _ := client.AddMagnet(magnetURI)
        // Wait for the download to complete
        <-t.GotInfo()
        t.DownloadAll()
        client.WaitAll()
        fi := t.Files()[0]
        fileName := filepath.Base(fi.Path())
        filePath := filepath.Join(".", fileName)

        os.Remove(filePath)
        if attempts < maxAttempts-1 {
            time.Sleep(5 * time.Second)
        }
        // Cleanup the torrent client
        client.Close()
    }
}

func sendRequests(config Config, stop chan struct{}) {
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
    // regex to find all URLs in response
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

    err = ioutil.WriteFile(filename, data, 0644)
    if err != nil {
        fmt.Println("Error writing config file:", err)
        return
    }
}
func generateWebSocketTraffic(config Config) {
    dialer := websocket.Dialer{}

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
