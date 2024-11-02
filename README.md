# anon-traffic-gen

A simple application to generate HTTP and WebSocket traffic. The code is designed and licensed solely for legitimate use, as a protective measure against MITM attacks. For instance, if you suspect network equipment compromise, this application can help obfuscate your traffic and mitigate eavesdropping risks.

## Description

This application is designed to generate HTTP and WebSocket traffic to a set of configurable URLs, using a variety of user agents. It can be used for various purposes, such as testing, load generation, or simulating user activity.

The application uses the Cobra library for the command-line interface, Viper for configuration management, and Gorilla WebSocket for WebSocket communication. The main traffic generation logic is handled in the `sendRequests()` function, which alternates between sending HTTP requests and generating WebSocket traffic, with a configurable rate limit.

## Architecture

1. **Configuration Management**: The application uses Viper to load and parse the configuration file, which specifies the URLs, user agents, and rate limit.

2. **Command-Line Interface**: Cobra is used to implement the command-line interface, allowing users to easily run the application with custom configuration options.

3. **Traffic Generation**: The `sendRequests()` function is the core of the application, managing the main traffic generation loop. It alternates between sending HTTP requests using the `sendHTTPRequests()` function and generating WebSocket traffic with the `generateWebSocketTraffic()` function, with a delay for rate limiting. The `downloadFileTorrent()` function downloads a file from a magnet URI using a torrent client, retrying the download up to a specified number of attempts, saving the file, and cleaning up the client, to hide parallel proxy connections.


## Code Structure

1. `Config` struct: Defines the structure of the configuration.
2. `rootCmd`: The main Cobra command that runs the application.
3. `initConfig()` and `loadConfig()`: Functions responsible for loading and parsing the configuration file.
4. `sendRequests()`: The main traffic generation loop.
   - `sendHTTPRequests()`: Sends HTTP GET requests to the configured URLs using the configured user agents.
   - `generateWebSocketTraffic()`: Establishes WebSocket connections to the configured URLs, sends a random message, and immediately closes the connection.
5. `main()`: The entry point of the application, which executes the `rootCmd`.
6. `downloadFileTorrent()` softly hides tunnel, vpn and proxy connections, via torrent ip connection

## Dependencies

The application relies on the following external libraries:

- [Cobra](https://github.com/spf13/cobra): For the command-line interface.
- [Viper](https://github.com/spf13/viper): For configuration management.
- [Gorilla WebSocket](https://github.com/gorilla/websocket): For WebSocket communication.
- [Anacrolix Torrent](https://github.com/anacrolix/torrent): For torrents IP
  
These dependencies are managed using Go modules.

## Configuration

The application uses a YAML configuration file, which can be specified using the `--config` flag. The default configuration file is `config.yaml`.

The configuration file should have the following structure:

```yaml
urls:
  - "https://example.com"
  - "https://another-example.com"
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8"
rate_limit: 1 # Requests per second
```

## Usage
### Installation Instructions

In Go, you typically don't need to manually install each dependency using `go get`. Instead, just run the `go build` command, which will automatically fetch and install any missing dependencies listed in your `go.mod` file.

```bash
go build
```


You might need to use `go get` for specific dependencies only if the automatic installation during `go build` fails. This can be useful when you're modifying the source code or setting up the project for the first time and want to ensure all dependencies are in place.

To install the required libraries for the `anon-traffic-gen` , you can use the following commands. Make sure you have Go installed on your system.

1. **Gorilla WebSocket**:
   ```bash
   go get github.com/gorilla/websocket
   ```

2. **Cobra**:
   ```bash
   go get github.com/spf13/cobra
   ```

3. **Viper**:
   ```bash
   go get github.com/spf13/viper
   ```

4. **YAML**:
   ```bash
   go get gopkg.in/yaml.v2
   ```

5. **Anacrolix Torrent**:
   ```bash
   go get github.com/anacrolix/torrent
   ```

### Potential Issues After Installation

After installing the libraries, you might run into some challenges that could affect how the application performs. There could be configuration issues that lead to instability, so it's a good idea to double-check your settings. 

In some cases, certain functionalities might be better implemented manually instead of relying entirely on the libraries, which could improve performance and give you more control. 

While the code works, there may be opportunities to enhance its readability and documentation, making it easier for others to understand and contribute. 

Additionally, the application might not always manage rate limiting perfectly, which could lead to unexpected behavior during traffic generation. The torrent functionality may also need some adjustments for optimal performance.

These aspects present a chance for improvement, and exploring the code could lead to valuable insights and enhancements!

To run the application, use the following command:

```
main --config=config.yaml
```

This will start the traffic generation and continue until you press to stop.

## Contributing

If you find any issues or have suggestions for improvements, feel free to open an issue or submit a pull request on the [anon-traffic-gen](https://github.com/YarBurArt/anon-traffic-gen/tree/main).

## License

This project is licensed under the [MIT License](https://github.com/YarBurArt/anon-traffic-gen/blob/main/LICENSE). Free use for peaceful purposes only.
