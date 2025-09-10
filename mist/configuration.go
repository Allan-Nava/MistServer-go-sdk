package mist_go

func defaultMistConfiguration() mistConfiguration {
	return mistConfiguration{
		Username: "",
		Password: "",
		BaseUrl:  "",
	}
}

type mistConfiguration struct {
	Username string
	Password string
	BaseUrl  string
}

func WithUsername(username string) func(mc *mistConfiguration) {
	return func(mc *mistConfiguration) {
		mc.Username = username
	}
}

func WithPassword(password string) func(mc *mistConfiguration) {
	return func(mc *mistConfiguration) {
		mc.Password = password
	}
}

func WithBaseURL(baseURL string) func(mc *mistConfiguration) {
	return func(mc *mistConfiguration) {
		mc.BaseUrl = baseURL
	}
}
