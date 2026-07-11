package main

import (
   "encoding/json"
   "io"
   "net/http"
   "sort"
   "strconv"
)

func average(values []float64) float64 {
   if len(values) == 0 {
      return 0
   }
   sum := 0.0
   for _, v := range values {
      sum += v
   }
   return sum / float64(len(values))
}

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
   Slug           string
   Name           string
   CreatedAt      string
   ContextLength  int64
   Intelligence   float64
   Coding         float64
   Agentic        float64
   BenefitAA      float64
   EloValues      []float64
   BenefitDA      float64
   InputPrice     float64
   OutputPrice    float64
   CacheReadPrice float64
   HasAA          bool
   HasDA          bool
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
      }

      // Benchmarks
      if b, ok := benchmarks[slug]; ok {
         if aa := getMap(b, "aa"); aa != nil {
            row.Intelligence = getFloat(aa, "intelligence_index")
            row.Coding = getFloat(aa, "coding_index")
            row.Agentic = getFloat(aa, "agentic_index")
            if row.Intelligence > 0 || row.Coding > 0 || row.Agentic > 0 {
               row.BenefitAA = average([]float64{row.Intelligence, row.Coding, row.Agentic})
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
                  row.BenefitDA = average(row.EloValues)
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

func FilterAndSort(rows []ModelData, benefitFlag string) []ResultRow {
   getBenefit := func(r ModelData) float64 {
      if benefitFlag == "aa" {
         return r.BenefitAA
      }
      return r.BenefitDA
   }

   var results []ResultRow
   for _, r := range rows {
      if benefitFlag == "aa" && !r.HasAA {
         continue
      }
      if benefitFlag == "da" && !r.HasDA {
         continue
      }
      if !r.HasPricing {
         continue
      }
      results = append(results, ResultRow{
         Model:   r,
         Benefit: getBenefit(r),
      })
   }

   // Sort descending by benefit
   sort.Slice(results, func(i, j int) bool {
      return results[i].Benefit > results[j].Benefit
   })

   return results
}
