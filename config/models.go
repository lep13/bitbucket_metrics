package config

type Config struct {
	BitbucketAccessToken string `json:"bitbucket_access_token"`
	MongoDBURI           string `json:"mongodb_uri"`
	Region               string `json:"region"`
}
