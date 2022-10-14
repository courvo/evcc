package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/vehicle/bluelink"
	"golang.org/x/oauth2"
)

// Bluelink is an api.Vehicle implementation
type Bluelink struct {
	*embed
	*bluelink.Provider
}

func init() {
	registry.AddWithStore("kia", NewKiaFromConfig)
	registry.AddWithStore("hyundai", NewHyundaiFromConfig)
}

// NewHyundaiFromConfig creates a new vehicle
func NewHyundaiFromConfig(factory store.Provider, other map[string]interface{}) (api.Vehicle, error) {
	settings := bluelink.Config{
		URI:               "https://prd.eu-ccapi.hyundai.com:8080",
		BasicToken:        "NmQ0NzdjMzgtM2NhNC00Y2YzLTk1NTctMmExOTI5YTk0NjU0OktVeTQ5WHhQekxwTHVvSzB4aEJDNzdXNlZYaG10UVI5aVFobUlGampvWTRJcHhzVg==",
		CCSPServiceID:     "6d477c38-3ca4-4cf3-9557-2a1929a94654",
		CCSPApplicationID: bluelink.HyundaiAppID,
		AuthClientID:      "64621b96-0f0d-11ec-82a8-0242ac130003",
		BrandAuthUrl:      "https://eu-account.hyundai.com/auth/realms/euhyundaiidm/protocol/openid-connect/auth?client_id=%s&scope=openid%%20profile%%20email%%20phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
	}

	return newBluelinkFromConfig("hyundai", factory, other, settings)
}

// NewKiaFromConfig creates a new vehicle
func NewKiaFromConfig(factory store.Provider, other map[string]interface{}) (api.Vehicle, error) {
	settings := bluelink.Config{
		URI:               "https://prd.eu-ccapi.kia.com:8080",
		BasicToken:        "ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		CCSPServiceID:     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
		CCSPApplicationID: bluelink.KiaAppID,
		AuthClientID:      "572e0304-5f8d-4b4c-9dd5-41aa84eed160",
		BrandAuthUrl:      "https://eu-account.kia.com/auth/realms/eukiaidm/protocol/openid-connect/auth?client_id=%s&scope=openid%%20profile%%20email%%20phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
	}

	return newBluelinkFromConfig("kia", factory, other, settings)
}

// newBluelinkFromConfig creates a new Vehicle
func newBluelinkFromConfig(brand string, factory store.Provider, other map[string]interface{}, settings bluelink.Config) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Language       string
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Language: "en",
		Expiry:   expiry,
		Cache:    interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)

	// var device string
	// deviceStore := factory(brand + ".tokens.bluelink.deviceid." + cc.User)
	// deviceStore.Load(&device)

	tokenStore := factory(brand + ".tokens.bluelink." + cc.User)
	identity := bluelink.NewIdentity(log, settings).WithStore(tokenStore)

	var token oauth2.Token
	if err := tokenStore.Load(&token); err == nil {
		identity.TokenSource = oauth.CachedTokenSource(tokenStore, oauth.RefreshTokenSource(&token, identity))
	} else {
		if err := identity.Login(cc.User, cc.Password, cc.Language); err != nil {
			return nil, err
		}
	}

	api := bluelink.NewAPI(log, settings.URI, identity)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v bluelink.Vehicle) string {
			return v.VIN
		},
	)

	if err != nil {
		return nil, err
	}

	v := &Bluelink{
		embed:    &cc.embed,
		Provider: bluelink.NewProvider(api, vehicle.VehicleID, cc.Expiry, cc.Cache),
	}

	return v, nil
}
