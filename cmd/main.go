package main

import (
	"strings"

	"github.com/wouldgo/name-dot-com-updater/pkg"
	"go.uber.org/zap"
)

func main() {
	opts := pkg.NewOptions()

	client, err := opts.Read()
	if err != nil {
		panic(err)
	}

	ipAddr, thisErr := pkg.Ipify()
	if thisErr != nil {
		panic(thisErr)
	}

	filteredDomains, filterDomainsErr := client.ListDomains()
	if filterDomainsErr != nil {
		panic(filterDomainsErr)
	}

	for aDomain, records := range filteredDomains {
		if aDomain != "" {

			recordsFromProvider, listRecordsErr := client.ListRecords(aDomain)
			if listRecordsErr != nil {

				panic(listRecordsErr)
			}

			for _, aRecordToManage := range records {
				selectedRecord, isRecordToUpdate := pkg.Find(recordsFromProvider, aRecordToManage)

				if isRecordToUpdate {
					client.Log.Info("updating record", zap.Any("record", selectedRecord), zap.String("ipAddr", ipAddr))

					err = client.UpdateRecord(selectedRecord, ipAddr)
					if err != nil {

						panic(err)
					}

					client.Log.Info("record updated", zap.Any("record", selectedRecord), zap.String("ipAddr", ipAddr))
				} else {

					host := strings.ReplaceAll(aRecordToManage, aDomain, "")
					if len(host) > 0 {
						host = host[0 : len(host)-1]
					}
					fqdn := aDomain + "."
					if host != "" {
						fqdn = host + "." + fqdn
					}

					newRecord := pkg.Record{
						Host:       host,
						DomainName: aDomain,
						Fqdn:       fqdn,
					}
					client.Log.Info("creating record", zap.Any("record", newRecord), zap.String("ipAddr", ipAddr))

					client.CreateRecord(newRecord, ipAddr)
					if err != nil {

						panic(err)
					}

					client.Log.Info("record created", zap.Any("record", newRecord), zap.String("ipAddr", ipAddr))
				}
			}
		}
	}
}
