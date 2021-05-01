package main

import (
	"fmt"
	"strings"

	"github.com/namedotcom/go/namecom"
)

func main() {
	ip, thisErr := ipify()
	if thisErr != nil {
		panic(thisErr)
	}

	Environment, err := New()
	if err != nil {
		panic(err)
	}

	filteredDomains, filterDomainsErr := Environment.ListDomains()
	if filterDomainsErr != nil {
		panic(filterDomainsErr)
	}

	for aDomain, records := range filteredDomains {
		if aDomain != "" {

			recordsFromProvider, listRecordsErr := Environment.ListRecords(&aDomain)
			if listRecordsErr != nil {

				panic(listRecordsErr)
			}

			for _, aRecordToManage := range records {
				selectedRecord, isRecordToUpdate := Find(recordsFromProvider, aRecordToManage+".")
				Host := strings.ReplaceAll(aRecordToManage, aDomain, "")
				if len(Host) > 0 {

					Host = Host[0 : len(Host)-1]
				}

				Fqdn := Host + "." + aDomain + "."

				if Host == "" {

					Fqdn = aDomain + "."
				}

				if isRecordToUpdate {
					fmt.Printf("Updating %v...\n\r", Fqdn)
					newRecord := &namecom.Record{
						ID:         selectedRecord.ID,
						Type:       "A",
						Host:       Host,
						DomainName: aDomain,
						Fqdn:       Fqdn,
						Answer:     ip,
						TTL:        300,
					}

					_, err = Environment.NameDotComClient.UpdateRecord(newRecord)
					if err != nil {

						panic(err)
					}

					fmt.Printf("%v updated\n\r", Fqdn)
				} else {
					fmt.Printf("Creating %v...\n\r", Fqdn)
					newRecord := &namecom.Record{
						Type:       "A",
						Host:       Host,
						DomainName: aDomain,
						Fqdn:       Fqdn,
						Answer:     ip,
						TTL:        300,
					}

					_, err = Environment.NameDotComClient.CreateRecord(newRecord)
					if err != nil {

						panic(err)
					}
					fmt.Printf("%v created\n\r", Fqdn)
				}
			}
		}
	}
}
