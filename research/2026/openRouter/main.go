package main

import (
   "flag"
   "fmt"
   "os"
)

func main() {
   maxOutput := flag.Float64("output", 0,
      "max $/M tokens (0 = no limit)")
   openOnly := flag.Bool("open", false,
      "only show models with open weights")
   flag.Parse()

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

   // Filter by max output price if specified
   if *maxOutput > 0 {
      var filtered []ModelData
      for _, r := range rows {
         if r.OutputPrice <= *maxOutput {
            filtered = append(filtered, r)
         }
      }
      rows = filtered
   }

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

   // Filter & sort
   results := FilterAndSort(rows)

   if len(results) == 0 {
      fmt.Fprintf(os.Stderr, "No models match the criteria\n")
      os.Exit(1)
   }

   // Print sort indicator
   fmt.Printf("Sorted by: max(intelligence, coding, agentic) descending\n")

   // Print human-readable output
   for _, r := range results {
      fmt.Println()
      fmt.Printf("Model: %s\n", r.Model.Name)
      fmt.Printf("Created: %s\n", r.Model.CreatedAt)
      fmt.Printf("Context length: %d tokens\n", r.Model.ContextLength)
      if r.Model.HfSlug != "" {
         fmt.Printf("Model weights: %s\n", r.Model.HfSlug)
      }
      fmt.Printf("Intelligence: %.1f\n", r.Model.Intelligence)
      fmt.Printf("Coding: %.1f\n", r.Model.Coding)
      fmt.Printf("Agentic: %.1f\n", r.Model.Agentic)
      fmt.Printf("Input price: $%.2f / M tokens\n", r.Model.InputPrice)
      fmt.Printf("Output price: $%.2f / M tokens\n", r.Model.OutputPrice)
      fmt.Printf("Cache read price: $%.2f / M tokens\n", r.Model.CacheReadPrice)
   }

   fmt.Fprintf(os.Stderr, "\nRanked %d models (out of %d total)\n",
      len(results), len(rows))
}
