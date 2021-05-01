package main

import (
	"strings"

	"github.com/namedotcom/go/namecom"
)

type EnvironmentConf struct {
	LogLevel         string
	Domains          []string
	NameDotComClient *namecom.NameCom
}

type Record struct {
	Name string
	ID   int32
}

func Find(slice []Record, val string) (Record, bool) {
	for _, item := range slice {
		if item.Name == val {
			return item, true
		}
	}
	return Record{}, false
}

func (e *EnvironmentConf) ListDomains() (map[string][]string, error) {
	ListDomainsRequest := namecom.ListDomainsRequest{}
	ListDomainsResponse, ListDomainsResponseErr := e.NameDotComClient.ListDomains(&ListDomainsRequest)

	if ListDomainsResponseErr != nil {

		return nil, ListDomainsResponseErr
	}

	toReturn := make(map[string][]string, len(ListDomainsResponse.Domains))
	for _, aDomain := range ListDomainsResponse.Domains {

		index := 0
		for _, aRecord := range e.Domains {

			if strings.HasSuffix(aRecord, aDomain.DomainName) {

				if toReturn[aDomain.DomainName] == nil {

					toReturn[aDomain.DomainName] = make([]string, len(e.Domains))
				}
				toReturn[aDomain.DomainName][index] = aRecord
				index += 1
			}
		}
	}

	return toReturn, nil
}

func (e *EnvironmentConf) ListRecords(aDomain *string) ([]Record, error) {

	ListRecordsRequest := namecom.ListRecordsRequest{
		DomainName: *aDomain,
	}

	ListRecordsResponse, ListRecordsErr := e.NameDotComClient.ListRecords(&ListRecordsRequest)

	if ListRecordsErr != nil {

		return nil, ListRecordsErr
	}
	recordsFromProvider := make([]Record, len(ListRecordsResponse.Records))
	for index, aRecord := range ListRecordsResponse.Records {
		if aRecord.Type == "A" {

			recordsFromProvider[index] = Record{
				Name: aRecord.Fqdn,
				ID:   aRecord.ID,
			}
		}
	}

	return recordsFromProvider, nil
}
