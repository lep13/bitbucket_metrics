package config

type Config struct {
    BitbucketUsername  string `json:"bitbucket_username"`
    BitbucketAppPassword string `json:"bitbucket_app_password"`
    MongoDBURI          string `json:"mongodb_uri"`
    Region              string `json:"region"`
}
