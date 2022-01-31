package cache

import (
	"fmt"

	"encoding/json"

	"github.com/stackql/go-openapistackql/openapistackql"

	"gopkg.in/yaml.v2"
)

type IMarshaller interface {
	Unmarshal(item *Item) error
	Marshal(item *Item) error
	GetKey() string
}

func GetMarshaller(key string) (IMarshaller, error) {
	switch key {
	case DefaultMarshallerKey:
		return &DefaultMarshaller{}, nil
	case RootMarshallerKey:
		return &RootDiscoveryMarshaller{}, nil
	case ServiceMarshallerKey:
		return &ServiceDiscoveryMarshaller{}, nil
	}
	return nil, fmt.Errorf("cannot find apt marshaller")
}

type DefaultMarshaller struct{}

func (dm *DefaultMarshaller) Unmarshal(item *Item) error {
	return json.Unmarshal(item.RawValue, &item.Value)
}

func (dm *DefaultMarshaller) Marshal(item *Item) error {
	return nil
}

func (dm *DefaultMarshaller) GetKey() string {
	return DefaultMarshallerKey
}

type RootDiscoveryMarshaller struct{}

func (dm *RootDiscoveryMarshaller) Unmarshal(item *Item) error {
	var err error
	var blob map[string]*openapistackql.Service
	err = json.Unmarshal(item.RawValue, &blob)
	if err != nil {
		return err
	}
	item.Value = blob
	return err
}

func (dm *RootDiscoveryMarshaller) Marshal(item *Item) error {
	var err error
	prov, ok := item.Value.(openapistackql.Provider)
	if !ok {
		return fmt.Errorf("cannot marshal root discovery doc of type = %T", prov)
	}
	item.RawValue, err = yaml.Marshal(&prov)
	return err
}

func (dm *RootDiscoveryMarshaller) GetKey() string {
	return RootMarshallerKey
}

type ServiceDiscoveryMarshaller struct{}

func (dm *ServiceDiscoveryMarshaller) Unmarshal(item *Item) error {
	blob, err := openapistackql.LoadServiceDocFromBytes(item.RawValue)
	item.Value = blob
	return err
}

func (dm *ServiceDiscoveryMarshaller) Marshal(item *Item) error {
	value, ok := item.Value.(*openapistackql.Service)
	if !ok {
		return fmt.Errorf("Cannot Marshal cache object of type: %T", item.Value)
	}
	bytes, err := value.MarshalJSON()
	item.RawValue = (json.RawMessage)(bytes)
	return err
}

func (dm *ServiceDiscoveryMarshaller) GetKey() string {
	return ServiceMarshallerKey
}
