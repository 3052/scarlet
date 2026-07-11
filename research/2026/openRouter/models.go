package main

import (
   "encoding/json"
   "io"
   "net/http"
   "sort"
   "strconv"
)

// ============================================================================
// Helpers
// ============================================================================

func getFloat(m map[string]interface{}, key string) float64 {
   if v, ok := m[key]; ok {
      switch n := v.(type) {
      case float64:
         return n
      case int64:
         return float64(n)
      }
   }
   return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
   if v, ok := m[key]; ok {
      switch n := v.(type) {
      case float64:
         return int64(n)
      case int64:
         return n
      }
   }
   return 0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
   if v, ok := m[key].(map[string]interface{}); ok {
      return v
   }
   return nil
}

func getString(m map[string]interface{}, key string) string {
   if v, ok := m[key].(string); ok {
      return v
   }
   return ""
}

func maxOf(values ...float64) float64 {
   if len(values) == 0 {
      return 0
   }
   m := values[0]
   for _, v := range values[1:] {
      if v > m {
         m = v
      }
   }
   return m
}

// ============================================================================
// Types
// ============================================================================

type APIResponse struct {
   Data struct {
      Models     []map[string]interface{}          `json:"models"`
      Benchmarks map[string]map[string]interface{} `json:"benchmarks"`
   } `json:"data"`
}

// ============================================================================
// Fetch & Parse
// ============================================================================

func FetchAndParse(url string) (*APIResponse, error) {
   resp, err := http.Get(url)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   body, err := io.ReadAll(resp.Body)
   if err != nil {
      return nil, err
   }
   var apiResp APIResponse
   if err := json.Unmarshal(body, &apiResp); err != nil {
      return nil, err
   }
   return &apiResp, nil
}

type ModelData struct {
   Slug           string
   Name           string
   CreatedAt      string
   ContextLength  int64
   HfSlug         string
   Intelligence   float64
   Coding         float64
   Agentic        float64
   BenefitAA      float64
   InputPrice     float64
   OutputPrice    float64
   CacheReadPrice float64
   HasAA          bool
   HasPricing     bool
}

func BuildModelData(apiResp *APIResponse) []ModelData {
   models := apiResp.Data.Models
   benchmarks := apiResp.Data.Benchmarks

   var rows []ModelData
   for _, m := range models {
      slug := getString(m, "permaslug")
      name := getString(m, "name")
      if slug == "" {
         continue
      }
      row := ModelData{
         Slug:          slug,
         Name:          name,
         CreatedAt:     getString(m, "created_at"),
         ContextLength: getInt64(m, "context_length"),
         HfSlug:        getString(m, "hf_slug"),
      }

      // Benchmarks
      if b, ok := benchmarks[slug]; ok {
         if aa := getMap(b, "aa"); aa != nil {
            row.Intelligence = getFloat(aa, "intelligence_index")
            row.Coding = getFloat(aa, "coding_index")
            row.Agentic = getFloat(aa, "agentic_index")
            if row.Intelligence > 0 || row.Coding > 0 || row.Agentic > 0 {
               row.BenefitAA = maxOf(row.Intelligence, row.Coding, row.Agentic)
               row.HasAA = true
            }
         }
      }

      // Pricing — convert $/token to $/M tokens for readability
      if ep := getMap(m, "endpoint"); ep != nil {
         if pricing := getMap(ep, "pricing"); pricing != nil {
            if p, err := strconv.ParseFloat(getString(pricing, "prompt"), 64); err == nil && p > 0 {
               row.InputPrice = p * 1e6
            }
            if p, err := strconv.ParseFloat(getString(pricing, "completion"), 64); err == nil && p > 0 {
               row.OutputPrice = p * 1e6
            }
            if p, err := strconv.ParseFloat(getString(pricing, "input_cache_read"), 64); err == nil && p > 0 {
               row.CacheReadPrice = p * 1e6
            }
            if row.InputPrice > 0 || row.OutputPrice > 0 {
               row.HasPricing = true
            }
         }
      }

      rows = append(rows, row)
   }
   return rows
}

type ResultRow struct {
   Model   ModelData
   Benefit float64
}

// ============================================================================
// Filter & Sort
// ============================================================================

func FilterAndSort(rows []ModelData) []ResultRow {
   var results []ResultRow
   for _, r := range rows {
      if !r.HasAA {
         continue
      }
      if !r.HasPricing {
         continue
      }
      results = append(results, ResultRow{
         Model:   r,
         Benefit: r.BenefitAA,
      })
   }

   // Sort descending by benefit
   sort.Slice(results, func(i, j int) bool {
      return results[i].Benefit > results[j].Benefit
   })

   return results
}
