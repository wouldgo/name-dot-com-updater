package pkg

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	namecom "github.com/namedotcom/go/v4/namecom"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrNoDomains          = errors.New("no domains specified")
	ErrMissingCredentials = errors.New("missing name dot com credentials")
	ErrStoreRecord        = errors.New("error in storing record changes")

	logEnvEnv, logEnvEnvSet     = os.LookupEnv("LOG_ENV")
	logLevelEnv, logLevelEnvSet = os.LookupEnv("LOG_LEVEL")

	domainsEnv, domainsEnvSet               = os.LookupEnv("DOMAINS")
	nameDotComCredEnv, nameDotComCredEnvSet = os.LookupEnv("NAME_DOT_COM_CONFS")

	nameDotComUsernameEnv, nameDotComUsernameEnvSet = os.LookupEnv("NAME_DOT_COM_USERNAME")
	nameDotComPasswordEnv, nameDotComPasswordEnvSet = os.LookupEnv("NAME_DOT_COM_PASSWORD")
)

type NameDotComCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type StringSlice struct {
	slice []string
}

func (s *StringSlice) String() string {
	return strings.Join(s.slice, ",")
}

func (s *StringSlice) Set(v string) error {
	s.slice = append(s.slice, v)
	return nil
}

func (s *StringSlice) Slice() []string {
	return s.slice
}

type Options struct {
	logEnv, logLevel *string

	nameDotComConf *NameDotComCredentials
	domains        *StringSlice
}

func NewOptions() Options {
	var logEnv, logLevel, nameDotComUsername, nameDotComPassword string

	var domains StringSlice

	flag.StringVar(&logEnv, "log-env", "development", "logging enviroment type: production, development (default: development)")
	flag.StringVar(&logLevel, "log-level", "debug", "logging level: info, debug, error, ... (default: debug)")

	flag.Var(&domains, "domain", "a domain to update")
	flag.StringVar(&nameDotComUsername, "username", "", "name.com username")
	flag.StringVar(&nameDotComPassword, "password", "", "name.com password")

	return Options{
		logEnv:   &logEnv,
		logLevel: &logLevel,
		nameDotComConf: &NameDotComCredentials{
			Username: nameDotComUsername,
			Password: nameDotComPassword,
		},
		domains: &domains,
	}
}

type Config struct {
	Log              *zap.Logger
	domains          []string
	nameDotComClient *namecom.NameCom
}

func (o *Options) Read() (Config, error) {
	flag.Parse()

	var zero Config
	if logEnvEnvSet {
		o.logEnv = &logEnvEnv
	}

	if logLevelEnvSet {
		o.logLevel = &logLevelEnv
	}

	logger, err := log(*o.logEnv, *o.logLevel)
	if err != nil {
		return zero, fmt.Errorf("error logger creation: %w", err)
	}

	if domainsEnvSet {
		logger.Debug("domains from environment variable", zap.String("domains", domainsEnv))
		for _, aDomain := range strings.Split(domainsEnv, ",") {

			o.domains.Set(aDomain)
		}
	}

	domains := o.domains.Slice()

	if len(domains) == 0 {
		return zero, ErrNoDomains
	}

	logger.Info("domains to handle", zap.Any("domains", domains))

	if nameDotComCredEnvSet {
		logger.Debug("credentials from enviroment object")
		nameDotComConfsStruct := NameDotComCredentials{}
		if err := json.Unmarshal([]byte(nameDotComCredEnv), &nameDotComConfsStruct); err != nil {
			return zero, fmt.Errorf("error decoding solver config: %v", err)
		}
		o.nameDotComConf.Username = nameDotComConfsStruct.Username
		o.nameDotComConf.Password = nameDotComConfsStruct.Password
	}

	if nameDotComUsernameEnvSet && nameDotComPasswordEnvSet {
		logger.Debug("credentials from single enviroment variables")
		o.nameDotComConf.Username = nameDotComUsernameEnv
		o.nameDotComConf.Password = nameDotComPasswordEnv
	}

	if o.nameDotComConf.Username == "" || o.nameDotComConf.Password == "" {
		return zero, ErrMissingCredentials
	}

	toReturn := Config{
		Log:              logger,
		domains:          domains,
		nameDotComClient: namecom.New(o.nameDotComConf.Username, o.nameDotComConf.Password),
	}
	logger.Info("name-dot-com-updater", zap.Strings("domains", toReturn.domains))

	return toReturn, nil
}

func (e *Config) ListDomains() (map[string][]string, error) {
	listDomainsRequest := namecom.ListDomainsRequest{}
	listDomainsResponse, listDomainsResponseErr := e.nameDotComClient.ListDomains(&listDomainsRequest)

	if listDomainsResponseErr != nil {
		return nil, fmt.Errorf("listing domains in error: %w", listDomainsResponseErr)
	}

	toReturn := make(map[string][]string, len(listDomainsResponse.Domains))
	for _, aDomain := range listDomainsResponse.Domains {

		index := 0
		for _, aRecord := range e.domains {
			if strings.HasSuffix(aRecord, aDomain.DomainName) {
				if toReturn[aDomain.DomainName] == nil {

					toReturn[aDomain.DomainName] = make([]string, len(e.domains))
				}
				toReturn[aDomain.DomainName][index] = aRecord
				index += 1
			}
		}
	}

	return toReturn, nil
}

type Record struct {
	Host, DomainName, Fqdn string
	ID                     int32
}

func (e *Config) ListRecords(aDomain string) ([]Record, error) {
	ListRecordsRequest := namecom.ListRecordsRequest{
		DomainName: aDomain,
	}

	listRecordsResponse, listRecordsErr := e.nameDotComClient.ListRecords(&ListRecordsRequest)

	if listRecordsErr != nil {

		return nil, fmt.Errorf("listing records in error: %w", listRecordsErr)
	}
	recordsFromProvider := make([]Record, len(listRecordsResponse.Records))
	for index, aRecord := range listRecordsResponse.Records {
		if aRecord.Type == "A" {

			recordsFromProvider[index] = Record{
				Host:       aRecord.Host,
				DomainName: aRecord.DomainName,
				Fqdn:       aRecord.Fqdn,
				ID:         aRecord.ID,
			}
		}
	}

	return recordsFromProvider, nil
}

func (e *Config) UpdateRecord(record Record, ipAddr string) error {
	newRecord := &namecom.Record{
		ID:         record.ID,
		Type:       "A",
		Host:       record.Host,
		DomainName: record.DomainName,
		Fqdn:       record.Fqdn,
		Answer:     ipAddr,
		TTL:        300,
	}
	_, err := e.nameDotComClient.UpdateRecord(newRecord)
	if err != nil {
		return errors.Join(ErrStoreRecord, fmt.Errorf("updating record %+v in error: %w", record, err))
	}
	return nil
}

func (e *Config) CreateRecord(record Record, ipAddr string) error {
	newRecord := &namecom.Record{
		Type:       "A",
		Host:       record.Host,
		DomainName: record.DomainName,
		Fqdn:       record.Fqdn,
		Answer:     ipAddr,
		TTL:        300,
	}

	_, err := e.nameDotComClient.CreateRecord(newRecord)
	if err != nil {
		return errors.Join(ErrStoreRecord, fmt.Errorf("creating record %+v in error: %w", record, err))
	}
	return nil
}

func log(env, level string) (*zap.Logger, error) {
	var encoder zapcore.Encoder

	if strings.EqualFold(env, "production") {
		config := zap.NewProductionEncoderConfig()
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		config := zap.NewDevelopmentEncoderConfig()
		encoder = zapcore.NewConsoleEncoder(config)
	}

	//writer := bufio.NewWriter(os.Stderr)
	ws := zapcore.AddSync(os.Stderr)

	logLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("error level string not valid: %w", err)
	}

	core := zapcore.NewCore(encoder, ws, logLevel)
	return zap.New(core), nil
}

func Ipify() (string, error) {
	resp, err := http.Get("https://api.ipify.org/?format=json")

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", readErr
	}

	type ipifyS struct {
		Ip string `json:"ip"`
	}

	Ipify := ipifyS{}
	if err := json.Unmarshal(body, &Ipify); err != nil {
		return "", fmt.Errorf("error decoding solver config: %v", err)
	}

	return Ipify.Ip, nil
}

func Find(slice []Record, val string) (Record, bool) {
	toCompare := val + "."
	for _, item := range slice {
		if item.Fqdn == toCompare {
			return item, true
		}
	}
	return Record{}, false
}
