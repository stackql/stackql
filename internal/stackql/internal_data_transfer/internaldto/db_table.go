package internaldto

import (
	"github.com/stackql/stackql/internal/stackql/constants"
)

var (
	_ DBTable = &standardDBTable{}
)

type DBTable interface {
	GetBaseName() string
	GetDiscoveryID() int
	GetHeirarchyIdentifiers() HeirarchyIdentifiers
	IsAnalytics() bool
	GetName() string
	GetNameStump() string
}

type standardDBTable struct {
	name        string
	nameStump   string
	baseName    string
	discoveryID int
	hIDs        HeirarchyIdentifiers
	namespace   string
}

func NewDBTable(name string, nameStump string, baseName string, discoveryID int, hIDs HeirarchyIdentifiers) DBTable {
	return newDBTable(name, nameStump, baseName, discoveryID, hIDs, "")
}

func NewDBTableAnalytics(name string, discoveryID int, hIDs HeirarchyIdentifiers) DBTable {
	return newDBTable(name, name, name, discoveryID, hIDs, constants.AnalyticsPrefix)
}

func newDBTable(
	name string,
	nameStump string,
	baseName string,
	discoveryID int,
	hIDs HeirarchyIdentifiers,
	namespace string,
) DBTable {
	return &standardDBTable{
		name:        name,
		nameStump:   nameStump,
		baseName:    baseName,
		discoveryID: discoveryID,
		hIDs:        hIDs,
		namespace:   namespace,
	}
}

func (dbt *standardDBTable) GetName() string {
	return dbt.name
}

func (dbt *standardDBTable) GetNameStump() string {
	return dbt.nameStump
}

func (dbt *standardDBTable) GetBaseName() string {
	return dbt.baseName
}

func (dbt *standardDBTable) GetDiscoveryID() int {
	return dbt.discoveryID
}

func (dbt *standardDBTable) GetHeirarchyIdentifiers() HeirarchyIdentifiers {
	return dbt.hIDs
}

func (dbt *standardDBTable) IsAnalytics() bool {
	return dbt.namespace == constants.AnalyticsPrefix
}
