package main

import (
   "flag"
   "fmt"
   "os"
)

func main() {
   benefitFlag := flag.String("b", "",
      "benefit source: 'aa' = average of intelligence/coding/agentic indices, 'da' = average ELO across categories (required)")
   maxOutput := flag.Float64("o", 0,
      "max output price in $/M tokens, e.g. 10 (0 = no limit)")
   flag.Parse()

   if *benefitFlag == "" || (*benefitFlag != "aa" && *benefitFlag != "da") {
      flag.Usage()
      os.Exit(1)
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

   // Filter & sort
   results := FilterAndSort(rows, *benefitFlag)

   if len(results) == 0 {
      fmt.Fprintf(os.Stderr, "No models match the criteria\n")
      os.Exit(1)
   }

   // Print human-readable output
   for i, r := range results {
      if i > 0 {
         fmt.Println()
      }
      fmt.Printf("Rank: %d\n", i+1)
      fmt.Printf("Model: %s\n", r.Model.Name)
      fmt.Printf("Created: %s\n", r.Model.CreatedAt)
      fmt.Printf("Context length: %d tokens\n", r.Model.ContextLength)
      if *benefitFlag == "aa" {
         fmt.Printf("Intelligence: %.1f\n", r.Model.Intelligence)
         fmt.Printf("Coding: %.1f\n", r.Model.Coding)
         fmt.Printf("Agentic: %.1f\n", r.Model.Agentic)
         fmt.Printf("Benefit (AA average): %.2f\n", r.Benefit)
      } else {
         fmt.Printf("ELO categories: %d\n", len(r.Model.EloValues))
         fmt.Printf("Benefit (DA average ELO): %.2f\n", r.Benefit)
      }
      fmt.Printf("Input price: $%.2f / M tokens\n", r.Model.InputPrice)
      fmt.Printf("Output price: $%.2f / M tokens\n", r.Model.OutputPrice)
      fmt.Printf("Cache read price: $%.2f / M tokens\n", r.Model.CacheReadPrice)
   }

   fmt.Fprintf(os.Stderr, "\nRanked %d models (out of %d total, -b=%s)\n",
      len(results), len(rows), *benefitFlag)
}
