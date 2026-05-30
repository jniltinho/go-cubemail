package commands

import (
	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
)

// ProvisionHandler implements the MS-ASPROV Provision command.
type ProvisionHandler struct{}

// Handle returns a policy-not-applied response (SOGo-compatible minimal provisioning).
func (h *ProvisionHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	_ = ctx
	_ = body
	resp := eas.ProvisionResponse{
		Status: eas.StatusSuccess,
		Policies: eas.PoliciesResponse{
			Policy: []eas.PolicyResponse{{
				PolicyType: eas.PolicyTypeWBXML,
				Status:     2,
			}},
		},
	}
	return wbxml.Marshal(resp)
}
