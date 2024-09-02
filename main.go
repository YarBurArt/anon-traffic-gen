package main

import (
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
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

func sendRequests(config Config, stop chan struct{}) {
    for {
        select {
        case <-stop:
            return
        default:
            for _, url := range config.URLs {
                for _, ua := range config.UserAgents {
                    // Create request
                    req, err := http.NewRequest(http.MethodGet, url, nil)
                    if err != nil {
                        fmt.Println("Error creating request:", err)
                        continue
                    }
                    req.Header.Set("User-Agent", ua)

                    // Send request and measure time
                    start := time.Now()
                    client := &http.Client{}
                    resp, err := client.Do(req)
                    elapsed := time.Since(start)

                    if err != nil {
                        fmt.Println("Error sending request:", err)
                        continue
                    }
                    defer resp.Body.Close()

                    // Print request details
                    fmt.Printf("Request: %s %s (User-Agent: %s)\n", http.MethodGet, url, ua)

                    // Print response details
                    fmt.Printf("Response: Status Code: %d, Elapsed Time: %s\n", resp.StatusCode, elapsed)

                    // Optional: Print response headers (if needed)
                    // for key, value := range resp.Header {
                    //     fmt.Printf("  Header: %s = %s\n", key, value)
                    // }

                    // Rate limiting logic here (optional)
                    time.Sleep(time.Second * time.Duration(config.RateLimit))
                }
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
