package config

type Config struct {
	BitbucketAccessToken string `json:"bitbucket_access_token"`
	MongoDBURI           string `json:"mongodb_uri"`
	Region               string `json:"region"`
	RepoURLTemplate      string `json:"repo_url_template"`
	CommitsURLTemplate   string `json:"commits_url_template"`
	CommitURLTemplate    string `json:"commit_url_template"`
	DiffstatURLTemplate  string `json:"diffstat_url_template"`
}
