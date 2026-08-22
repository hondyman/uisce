package mdm

import (
	"time"

	"github.com/google/uuid"
)

type DownstreamSyncRequest struct {
	TenantID       uuid.UUID              `json:"tenantId"`
	BOID           uuid.UUID              `json:"boId"`
	EntitySID      string                 `json:"entitySid"`
	GoldAttributes map[string]interface{} `json:"goldAttributes"`
	MutationType   string                 `json:"mutationType"` // UPSERT, DELETE
	KnowledgeTime  time.Time              `json:"knowledgeTime"`
}

type BindingTargetDescriptor struct {
	BindingID       uuid.UUID `json:"bindingId"`
	TargetName      string    `json:"targetName"`
	DeliveryChannel string    `json:"deliveryChannel"`
	EndpointURL     string    `json:"endpointUrl,omitempty"`
}
