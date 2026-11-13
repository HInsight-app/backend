package request

type GenerateInsightRequest struct {
	TriggerContext string `json:"trigger_context" validate:"required,oneof=pre_decision post_evaluation"`
}
