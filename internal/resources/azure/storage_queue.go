package azure

import (
	"fmt"
	"strings"

	"github.com/infracost/infracost/internal/resources"
	"github.com/infracost/infracost/internal/schema"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// StorageQueue struct represents Azure Queue Storage.
//
// Resource information: https://azure.microsoft.com/en-gb/pricing/details/storage/queues/
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/storage/queues/#pricing
type StorageQueue struct {
	Address                string
	Region                 string
	AccountReplicationType string
}

// CoreType returns the name of this resource type
func (r *StorageQueue) CoreType() string {
	return "StorageQueue"
}

// UsageSchema defines a list which represents the usage schema of StorageQueue.
func (r *StorageQueue) UsageSchema() []*schema.UsageItem {
	return []*schema.UsageItem{}
}

// PopulateUsage parses the u schema.UsageData into the StorageQueue.
// It uses the `infracost_usage` struct tags to populate data into the StorageQueue.
func (r *StorageQueue) PopulateUsage(u *schema.UsageData) {
	resources.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a schema.Resource from a valid StorageQueue struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *StorageQueue) BuildResource() *schema.Resource {
	if !r.isReplicationTypeSupported() {
		log.Warnf("Skipping resource %s. Storage queues don't support %s redundancy", r.Address, r.AccountReplicationType)
		return nil
	}

	costComponents := []*schema.CostComponent{
		r.dataStorageCostComponent(),
	}
	costComponents = append(costComponents, r.operationsCostComponents()...)
	costComponents = append(costComponents, r.geoReplicationDataTransferCostComponents()...)

	return &schema.Resource{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *StorageQueue) isReplicationTypeSupported() bool {
	return contains([]string{"LRS", "ZRS", "GRS", "RA-GRS", "GZRS", "RA-GZRS"}, strings.ToUpper(r.AccountReplicationType))
}

func (r *StorageQueue) dataStorageCostComponent() *schema.CostComponent {
	return &schema.CostComponent{
		Name:            "Capacity",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: nil,
		ProductFilter: &schema.ProductFilter{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*schema.AttributeFilter{
				{Key: "productName", Value: strPtr("Queues v2")},
				{Key: "skuName", Value: strPtr(fmt.Sprintf("Standard %s", strings.ToUpper(r.AccountReplicationType)))},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Data Stored", strings.ToUpper(r.AccountReplicationType)))},
			},
		},
		PriceFilter: &schema.PriceFilter{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
	}
}

func (r *StorageQueue) operationsCostComponents() []*schema.CostComponent {
	costComponents := []*schema.CostComponent{}

	if !contains([]string{"GZRS", "RA-GZRS"}, strings.ToUpper(r.AccountReplicationType)) {
		costComponents = append(costComponents, &schema.CostComponent{
			Name:            "Class 1 operations",
			Unit:            "10k operations",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: nil,
			ProductFilter: &schema.ProductFilter{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "productName", Value: strPtr("Queues v2")},
					{Key: "skuName", Value: strPtr(fmt.Sprintf("Standard %s", strings.ToUpper(r.AccountReplicationType)))},
					{Key: "meterName", ValueRegex: regexPtr("Class 1 Operations$")},
				},
			},
			PriceFilter: &schema.PriceFilter{
				PurchaseOption:   strPtr("Consumption"),
				StartUsageAmount: strPtr("0"),
			},
		})
	}

	costComponents = append(costComponents, &schema.CostComponent{
		Name:            "Class 2 operations",
		Unit:            "10k operations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: nil,
		ProductFilter: &schema.ProductFilter{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*schema.AttributeFilter{
				{Key: "productName", Value: strPtr("Queues v2")},
				{Key: "skuName", Value: strPtr(fmt.Sprintf("Standard %s", strings.ToUpper(r.AccountReplicationType)))},
				{Key: "meterName", ValueRegex: regexPtr("Class 2 Operations$")},
			},
		},
		PriceFilter: &schema.PriceFilter{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
	})

	return costComponents
}

func (r *StorageQueue) geoReplicationDataTransferCostComponents() []*schema.CostComponent {
	if contains([]string{"LRS", "ZRS"}, strings.ToUpper(r.AccountReplicationType)) {
		return []*schema.CostComponent{}
	}

	return []*schema.CostComponent{
		{
			Name:            "Geo-replication data transfer",
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: nil,
			ProductFilter: &schema.ProductFilter{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "productName", Value: strPtr("Storage - Bandwidth")},
					{Key: "skuName", Value: strPtr("Geo-Replication v2")},
					{Key: "meterName", Value: strPtr("Geo-Replication v2 Data Transfer")},
				},
			},
			PriceFilter: &schema.PriceFilter{
				PurchaseOption:   strPtr("Consumption"),
				StartUsageAmount: strPtr("0"),
			},
		},
	}
}
