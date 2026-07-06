package main

import (
   _ "embed"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "net/http"
   "os"
   "path/filepath"
   "strings"
)

const serverAddress = "localhost:8080"

const sessionFileName = "session.json"

//go:embed favicon.svg
var faviconSVG string

//go:embed index.html
var indexHTML string

//go:embed style.css
var styleCSS string

func main() {
   log.SetFlags(log.Ltime)

   apiKeyFlag := flag.String("api-key", "", "Update the API key in your config")
   apiUrlFlag := flag.String("api-url", "", "Update the API URL in your config (expects an OpenAI-compatible chat completions endpoint)")
   modelFlag := flag.String("model", "", "Update the Model name in your config")
   serveFlag := flag.Bool("serve", false, "Start the local chatbot server")

   flag.Parse()

   // Strictly enforce exactly one flag at a time
   if flag.NFlag() != 1 || flag.NArg() > 0 {
      fmt.Println("Error: You must provide exactly one valid flag at a time.")
      flag.Usage()
      os.Exit(1)
   }

   if err := run(apiKeyFlag, apiUrlFlag, modelFlag, *serveFlag); err != nil {
      log.Fatal(err)
   }
}

func run(apiKeyFlag, apiUrlFlag, modelFlag *string, serve bool) error {
   headerHTML, footerHTML, found := strings.Cut(indexHTML, "<!-- CHAT_CONTENT -->")
   if !found {
      return fmt.Errorf("error: index.html is missing the <!-- CHAT_CONTENT --> marker")
   }

   configDir, err := os.UserConfigDir()
   if err != nil {
      return fmt.Errorf("error getting user config directory: %w", err)
   }

   appConfigDir := filepath.Join(configDir, "chatbot")
   configFilePath := filepath.Join(appConfigDir, "config.json")

   // Start with an empty config
   var cfg AppConfig

   // Try to load existing config
   if data, err := os.ReadFile(configFilePath); err == nil {
      json.Unmarshal(data, &cfg)
   }

   // Update config ONLY if flags were explicitly provided by the user in the CLI
   updated := false
   flag.Visit(func(f *flag.Flag) {
      switch f.Name {
      case "api-key":
         cfg.APIKey = *apiKeyFlag
         updated = true
      case "api-url":
         cfg.APIURL = *apiUrlFlag
         updated = true
      case "model":
         cfg.Model = *modelFlag
         updated = true
      }
   })

   if updated {
      if err := os.MkdirAll(appConfigDir, 0700); err != nil {
         return fmt.Errorf("error creating config directory: %w", err)
      }
      configData, _ := json.MarshalIndent(cfg, "", "  ")
      if err := os.WriteFile(configFilePath, configData, 0600); err != nil {
         return fmt.Errorf("error writing config file: %w", err)
      }
      log.Println("Configuration updated successfully.")
   }

   // If the user did not explicitly request to start the server, exit here
   if !serve {
      return nil
   }

   // Ensure all required configuration fields are present before starting
   if cfg.APIKey == "" {
      return fmt.Errorf("api key not found; please run with '-api-key YOUR_KEY'")
   }
   if cfg.APIURL == "" {
      return fmt.Errorf("api url not found; please run with '-api-url YOUR_URL'")
   }
   if cfg.Model == "" {
      return fmt.Errorf("model not found; please run with '-model YOUR_MODEL'")
   }

   http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Content-Type", "text/css")
      fmt.Fprint(w, styleCSS)
   })

   http.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Content-Type", "image/svg+xml")
      fmt.Fprint(w, faviconSVG)
   })

   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      if err := handleRoot(w, r, cfg, headerHTML, footerHTML); err != nil {
         log.Printf("Handler error: %v", err)
      }
   })

   log.Printf("Starting local server at http://%s - Press Ctrl+C to stop", serverAddress)
   return http.ListenAndServe(serverAddress, nil)
}

type AppConfig struct {
   APIKey string `json:"api_key"`
   APIURL string `json:"api_url"`
   Model  string `json:"model"`
}
