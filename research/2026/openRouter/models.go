package main

import (
   "encoding/json"
   "io"
   "math"
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

func median(values []float64) float64 {
   if len(values) == 0 {
      return 0
   }
   sorted := make([]float64, len(values))
   copy(sorted, values)
   sort.Float64s(sorted)
   n := len(sorted)
   if n%2 == 1 {
      return sorted[n/2]
   }
   return (sorted[n/2-1] + sorted[n/2]) / 2
}

// ============================================================================
// Types
// ============================================================================

type APIResponse struct {
   Data struct {
      Models     []map[string]interface{}          `json:"models"`
      Benchmarks map[string]map[string]interface{} `json:"benchmarks"`
      Analytics  map[string]map[string]interface{} `json:"analytics"`
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
   Slug             string
   Name             string
   CreatedAt        string
   Intelligence     float64
   Coding           float64
   Agentic          float64
   BenefitAA        float64
   EloValues        []float64
   BenefitDA        float64
   InputPrice       float64
   OutputPrice      float64
   CacheReadPrice   float64
   PromptTokens     float64
   CompletionTokens float64
   CachedTokens     float64
   CostPerM         float64
   HasAA            bool
   HasDA            bool
   HasPricing       bool
}

func BuildModelData(apiResp *APIResponse) []ModelData {
   models := apiResp.Data.Models
   benchmarks := apiResp.Data.Benchmarks
   analytics := apiResp.Data.Analytics

   var rows []ModelData
   for _, m := range models {
      slug := getString(m, "permaslug")
      name := getString(m, "name")
      if slug == "" {
         continue
      }
      row := ModelData{
         Slug:      slug,
         Name:      name,
         CreatedAt: getString(m, "created_at"),
      }

      // Benchmarks
      if b, ok := benchmarks[slug]; ok {
         if aa := getMap(b, "aa"); aa != nil {
            row.Intelligence = getFloat(aa, "intelligence_index")
            row.Coding = getFloat(aa, "coding_index")
            row.Agentic = getFloat(aa, "agentic_index")
            if row.Intelligence > 0 || row.Coding > 0 || row.Agentic > 0 {
               row.BenefitAA = row.Intelligence + row.Coding + row.Agentic
               row.HasAA = true
            }
         }
         if da := getMap(b, "da"); da != nil {
            if eloCat, ok := da["elo_by_category"].(map[string]interface{}); ok {
               for _, v := range eloCat {
                  if f := getFloat(map[string]interface{}{"v": v}, "v"); f > 0 {
                     row.EloValues = append(row.EloValues, f)
                  }
               }
               if len(row.EloValues) > 0 {
                  row.BenefitDA = median(row.EloValues)
                  row.HasDA = true
               }
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

      // Analytics
      if a, ok := analytics[slug]; ok {
         row.PromptTokens = getFloat(a, "total_prompt_tokens")
         row.CompletionTokens = getFloat(a, "total_completion_tokens")
         row.CachedTokens = getFloat(a, "total_native_tokens_cached")
      }

      // Cost: usage-weighted average $/M tokens
      totalTokens := row.PromptTokens + row.CompletionTokens + row.CachedTokens
      if totalTokens > 0 && row.HasPricing {
         weightedCost := row.InputPrice*row.PromptTokens +
            row.OutputPrice*row.CompletionTokens +
            row.CacheReadPrice*row.CachedTokens
         row.CostPerM = weightedCost / totalTokens
      }

      rows = append(rows, row)
   }
   return rows
}

type ResultRow struct {
   Model       ModelData
   Benefit     float64
   BenefitNorm float64
   CostNorm    float64
   Value       float64
}

// ============================================================================
// Filter & Compute
// ============================================================================

func FilterAndCompute(rows []ModelData, benefitFlag string) []ResultRow {
   // Filter
   var filtered []ModelData
   for _, r := range rows {
      if benefitFlag == "aa" && !r.HasAA {
         continue
      }
      if benefitFlag == "da" && !r.HasDA {
         continue
      }
      if !r.HasPricing || r.CostPerM <= 0 {
         continue
      }
      filtered = append(filtered, r)
   }

   // Compute min/max
   var benefitMin, benefitMax, costMin, costMax float64
   benefitMin = math.Inf(1)
   benefitMax = math.Inf(-1)
   costMin = math.Inf(1)
   costMax = math.Inf(-1)

   getBenefit := func(r ModelData) float64 {
      if benefitFlag == "aa" {
         return r.BenefitAA
      }
      return r.BenefitDA
   }

   for _, r := range filtered {
      b := getBenefit(r)
      if b < benefitMin {
         benefitMin = b
      }
      if b > benefitMax {
         benefitMax = b
      }
      if r.CostPerM < costMin {
         costMin = r.CostPerM
      }
      if r.CostPerM > costMax {
         costMax = r.CostPerM
      }
   }

   // Compute value = benefit_norm - cost_norm
   var results []ResultRow
   for _, r := range filtered {
      b := getBenefit(r)
      var benefitNorm, costNorm float64
      if benefitMax > benefitMin {
         benefitNorm = (b - benefitMin) / (benefitMax - benefitMin)
      }
      if costMax > costMin {
         costNorm = (r.CostPerM - costMin) / (costMax - costMin)
      }
      value := benefitNorm - costNorm
      results = append(results, ResultRow{
         Model:       r,
         Benefit:     b,
         BenefitNorm: benefitNorm,
         CostNorm:    costNorm,
         Value:       value,
      })
   }

   // Sort descending by value
   sort.Slice(results, func(i, j int) bool {
      return results[i].Value > results[j].Value
   })

   return results
}
