package main

import (
   "flag"
   "fmt"
   "os"
)

var sortKeys = []string{"elo", "intelligence", "coding", "agentic"}

func main() {
   openOnly := flag.Bool("open", false,
      "only show models with open weights")
   imageOnly := flag.Bool("image", false,
      "only show models that accept image input")
   export := flag.Bool("export", false,
      "fetch models and write a file for every sort key")
   flag.Parse()

   // No flags -> do nothing.
   if !*export {
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

   // Filter by image input if specified
   if *imageOnly {
      var filtered []ModelData
      for _, r := range rows {
         if r.HasImage {
            filtered = append(filtered, r)
         }
      }
      rows = filtered
   }

   // For each sort key, filter+sort and write its own file.
   for _, key := range sortKeys {
      results := FilterAndSort(rows, key)
      fname := "models_" + key + ".txt"
      f, err := os.Create(fname)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", fname, err)
         os.Exit(1)
      }

      fmt.Fprintf(f, "Sorted by: %s descending\n", key)
      if *openOnly {
         fmt.Fprintln(f, "Filter: open weights only")
      }
      if *imageOnly {
         fmt.Fprintln(f, "Filter: image input only")
      }

      if len(results) == 0 {
         fmt.Fprintln(f, "No models match the criteria")
         f.Close()
         fmt.Fprintf(os.Stderr, "Wrote %s (0 models)\n", fname)
         continue
      }

      for _, r := range results {
         fmt.Fprintln(f)
         fmt.Fprintf(f, "Model: %s\n", r.Name)
         fmt.Fprintf(f, "Created: %s\n", r.CreatedAt)
         fmt.Fprintf(f, "Context length: %d tokens\n", r.ContextLength)
         if r.HfSlug != "" {
            fmt.Fprintf(f, "Model weights: %s\n", r.HfSlug)
         }
         if r.HasImage {
            fmt.Fprintln(f, "Image input: yes")
         }
         fmt.Fprintf(f, "Arena ELO: %.1f\n", r.Elo)
         fmt.Fprintf(f, "Intelligence: %.1f\n", r.Intelligence)
         fmt.Fprintf(f, "Coding: %.1f\n", r.Coding)
         fmt.Fprintf(f, "Agentic: %.1f\n", r.Agentic)
         fmt.Fprintf(f, "Input price: $%.2f / M tokens\n", r.InputPrice)
         fmt.Fprintf(f, "Output price: $%.2f / M tokens\n", r.OutputPrice)
         fmt.Fprintf(f, "Cache read price: $%.2f / M tokens\n", r.CacheReadPrice)
      }

      fmt.Fprintf(f, "\nRanked %d models (out of %d total)\n",
         len(results), len(rows))
      f.Close()
      fmt.Fprintf(os.Stderr, "Wrote %s (%d models)\n", fname, len(results))
   }

   fmt.Fprintf(os.Stderr, "\nProcessed %d total models across %d sort keys\n",
      len(rows), len(sortKeys))
}
