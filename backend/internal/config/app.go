package config

// appConfig — общие параметры процесса. Оба значения попадают в каждую
// запись лога (атрибуты service и env) и в ответ /health/live.
type appConfig struct {
	Env         string // local, staging, production
	ServiceName string
}

func loadAppConfig(lookup LookupFunc) appConfig {
	return appConfig{
		Env:         getString(lookup, "APP_ENV", envLocal),
		ServiceName: getString(lookup, "SERVICE_NAME", "netquest-api"),
	}
}
