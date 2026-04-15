package awsce

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/reusaymane/platformx-finops/data-collector/internal/db"
	"go.uber.org/zap"
)

type AWSCostExplorer struct {
	client *costexplorer.Client
	logger *zap.Logger
	region string
}

func New(region string, logger *zap.Logger) (*AWSCostExplorer, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &AWSCostExplorer{
		client: costexplorer.NewFromConfig(cfg),
		logger: logger,
		region: region,
	}, nil
}

func (a *AWSCostExplorer) Name() string { return "aws_cost_explorer" }

func (a *AWSCostExplorer) Collect(ctx context.Context) ([]db.CostRecord, error) {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -1).Format("2006-01-02")
	end := now.Format("2006-01-02")

	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &start,
			End:   &end,
		},
		Granularity: types.GranularityHourly,
		GroupBy: []types.GroupDefinition{
			{Type: types.GroupDefinitionTypeDimension, Key: strPtr("SERVICE")},
			{Type: types.GroupDefinitionTypeTag, Key: strPtr("kubernetes_namespace")},
		},
		Metrics: []string{"UnblendedCost"},
	}

	resp, err := a.client.GetCostAndUsage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("aws cost explorer error: %w", err)
	}

	var records []db.CostRecord
	for _, result := range resp.ResultsByTime {
		t, _ := time.Parse("2006-01-02T15:04:05Z", *result.TimePeriod.Start)
		for _, group := range result.Groups {
			cost := 0.0
			if m, ok := group.Metrics["UnblendedCost"]; ok {
				cost, _ = strconv.ParseFloat(*m.Amount, 64)
			}
			if cost == 0 {
				continue
			}

			namespace := "default"
			service := "unknown"
			if len(group.Keys) > 0 {
				service = group.Keys[0]
			}
			if len(group.Keys) > 1 {
				namespace = group.Keys[1]
			}

			records = append(records, db.CostRecord{
				Time:        t,
				Namespace:   namespace,
				Service:     service,
				Environment: "prod",
				Team:        "unknown",
				CostUSD:     cost,
				Source:      "aws_cost_explorer",
			})
		}
	}

	a.logger.Info("collected from AWS Cost Explorer",
		zap.Int("records", len(records)),
		zap.String("period", start+" → "+end),
	)
	return records, nil
}

func strPtr(s string) *string { return &s }
