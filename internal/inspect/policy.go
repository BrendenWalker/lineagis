package inspect

import (
	"context"
	"encoding/json"

	"github.com/BrendenWalker/lineagis/internal/apiclient"
)

func policyDocument(ctx context.Context, api *apiclient.Client, namespace string) json.RawMessage {
	if api == nil {
		return nil
	}
	p, err := api.GetPolicy(ctx, namespace)
	if err != nil || p == nil {
		return nil
	}
	return p.Document
}
