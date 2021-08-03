package types

type AuroraConfig struct {
	Username   string
	Password   string
	UserId     int
	ActivityId int
}

type EmailConfig struct {
	Domain string
	ApiKey string
	From   string
	To     []string
}
