package main

import (
   "flag"
   "fmt"
   "os"
)

func main() {
   openOnly := flag.Bool("open", false,
      "only show models with open weights")
   sortBy := flag.String("sort", "",
      "sort key (required): elo, intelligence, coding, agentic")
   flag.Parse()

   // Validate -sort
   switch *sortBy {
   case "elo", "intelligence", "coding", "agentic":
      // ok
   default:
      flag.Usage()
      return
   }

   // Fetch
   url := "https://openrouter.ai/api/frontend/v1/models/find?active=true"
   fmt.Fprintf(os.Stderr, "Fetching from %s ...\n", url)
   apiResp, err := FetchAndParse(url)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error: %v\n", err)
      os.Exit(1)
   }

   // Build model data
   rows := BuildModelData(apiResp)

   // Filter by open weights if specified
   if *openOnly {
      var filtered []ModelData
      for _, r := range rows {
         if r.HfSlug != "" {
            filtered = append(filtered, r)
         }
      }
      rows = filtered
   }

   // Filter & sort by the chosen key
   results := FilterAndSort(rows, *sortBy)

   if len(results) == 0 {
      fmt.Fprintln(os.Stderr, "No models match the criteria")
      os.Exit(1)
   }

   // Print sort indicator
   fmt.Printf("Sorted by: %s descending\n", *sortBy)

   // Print human-readable output
   for _, r := range results {
      fmt.Println()
      fmt.Printf("Model: %s\n", r.Name)
      fmt.Printf("Created: %s\n", r.CreatedAt)
      fmt.Printf("Context length: %d tokens\n", r.ContextLength)
      if r.HfSlug != "" {
         fmt.Printf("Model weights: %s\n", r.HfSlug)
      }
      fmt.Printf("Arena ELO: %.1f\n", r.Elo)
      fmt.Printf("Intelligence: %.1f\n", r.Intelligence)
      fmt.Printf("Coding: %.1f\n", r.Coding)
      fmt.Printf("Agentic: %.1f\n", r.Agentic)
      fmt.Printf("Input price: $%.2f / M tokens\n", r.InputPrice)
      fmt.Printf("Output price: $%.2f / M tokens\n", r.OutputPrice)
      fmt.Printf("Cache read price: $%.2f / M tokens\n", r.CacheReadPrice)
   }

   fmt.Fprintf(os.Stderr, "\nRanked %d models (out of %d total)\n",
      len(results), len(rows))
}
