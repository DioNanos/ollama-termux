package server

import (
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/format"
)

// termuxModelRecommendations is the curated recommendation list served on
// Termux from the /api/experimental/model-recommendations endpoint.
//
// Upstream 0.30.x moved the launcher model picker onto this server endpoint
// (cmd/launch requestRecommendations), which fetches the live ollama.com list
// via the recommendations cache. The cmd/launch hardcoded fallback
// (termuxRecommendedModels) only applies when that endpoint fails, so the
// Termux override has to live here to actually reach the picker.
//
// Cloud-first, ordered by coding/agentic benchmark standing (June 2026), with
// two phone-sized local models kept at the bottom as an offline fallback.
// Mirrors cmd/launch termuxRecommendedModels.
var termuxModelRecommendations = []api.ModelRecommendation{
	{Model: "deepseek-v4-pro:cloud", Description: "Frontier coding and reasoning with a 1M-token context (LiveCodeBench/Codeforces leader)", ContextLength: 1_048_576, MaxOutputTokens: 65_536},
	{Model: "kimi-k2.7-code:cloud", Description: "Coding-specialized long-horizon agent, successor to Kimi K2.6", ContextLength: 262_144, MaxOutputTokens: 262_144},
	{Model: "minimax-m3:cloud", Description: "Fast agentic coding with a 512K context (top open-weight SWE-Bench Pro)", ContextLength: 524_288, MaxOutputTokens: 131_072},
	{Model: "glm-5.1:cloud", Description: "Strong structured code generation and agentic web development", ContextLength: 202_752, MaxOutputTokens: 131_072},
	{Model: "gemini-3-flash-preview:cloud", Description: "Fast, low-cost multimodal reasoning and tool use with a 1M-token context", ContextLength: 1_048_576, MaxOutputTokens: 65_536},
	{Model: "qwen3.5:4b", Description: "Local offline fallback for coding, reasoning, and visual understanding", VRAMBytes: 11 * format.GigaByte},
	{Model: "gemma4:e4b", Description: "Local offline fallback for reasoning and code generation", VRAMBytes: 16 * format.GigaByte},
}
