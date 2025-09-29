package events

type EventType string

type ContentType string

const (
	LLMChunk EventType = "llm_chunk"
	Error    EventType = "error"
)

const (
	ContentThought              ContentType = "thought"
	ContentOrchestratorMessage  ContentType = "orchestrator_message"
	ContentNova3ObserveThink    ContentType = "nova3_observe_think"
	ContentLoomiActionNote      ContentType = "loomi_action_note"
	ContentLoomiKnowledge       ContentType = "loomi_knowledge"
	ContentLoomiPersona         ContentType = "loomi_persona"
	ContentLoomiResonant        ContentType = "loomi_resonant"
	ContentLoomiHitpoint        ContentType = "loomi_hitpoint"
	ContentLoomiXHSPost         ContentType = "loomi_xhs_post"
	ContentLoomiOrchestrator    ContentType = "loomi_orchestrator"
	ContentLoomiConcierge       ContentType = "loomi_concierge"
	ContentLoomiWebSearch       ContentType = "loomi_websearch"
	ContentLoomiTikTokScript    ContentType = "loomi_tiktok_script"
	ContentLoomiWeChatArticle   ContentType = "loomi_wechat_article"
	ContentLoomiRevision        ContentType = "loomi_revision"
	ContentLoomiBrandAnalysis   ContentType = "loomi_brand_analysis"
	ContentLoomiContentAnalysis ContentType = "loomi_content_analysis"
	ContentSystemMessage        ContentType = "system_message"
	ContentBillingSummary       ContentType = "billing_summary"
	ContentAgentOtherMessage    ContentType = "agent_other_message"
	ContentNova3ZhipuWebsearch  ContentType = "nova3_zhipu_websearch"
	ContentNova3Websearch       ContentType = "nova3_websearch"
	// Concierge-specific content types (to mirror Python behavior)
	ContentConciergeMessage   ContentType = "concierge_message"
	ContentConciergeWebsearch ContentType = "concierge_websearch"
	ContentLoomiPlanConcierge ContentType = "loomi_plan_concierge"
)

type StreamEvent struct {
	Type    EventType      `json:"type"`
	Content ContentType    `json:"content_type"`
	Data    any            `json:"data"`
	Meta    map[string]any `json:"meta,omitempty"`
}
