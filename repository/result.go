package repository

type ResultFile struct {
	Repos []RepoUpdateResult `json:"repos"`
}

type RepoUpdateResult struct {
	Owner       string             `json:"owner"`
	Repo        string             `json:"repo"`
	Error       *string            `json:"error"`
	PullRequest *PullRequestResult `json:"pr"`
}

type PullRequestResult struct {
	Number int    `json:"number"`
	NodeID string `json:"nodeId"`
	URL    string `json:"url"`
}
