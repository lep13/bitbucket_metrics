package bitbucket

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	// "github.com/lep13/bitbucket_metrics/config"
	db "github.com/lep13/bitbucket_metrics/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockHTTPClient simulates the behavior of the HTTP client.
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) != nil {
		return args.Get(0).(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockCollection is a mock type for the mongo.Collection used for testing.
type MockCollection struct {
	mock.Mock
}

func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func TestFetchRepositories(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"values": [
				{"name": "repo1", "slug": "repo1", "project": {"name": "Project1"}},
				{"name": "repo2", "slug": "repo2", "project": {"name": "Project2"}}
			]
		}`)),
	}, nil).Once()

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	repos, err := fetchRepositories("fake_token")
	assert.NoError(t, err)
	assert.Len(t, repos, 2)
	assert.Equal(t, "repo1", repos[0].Name)
	assert.Equal(t, "repo2", repos[1].Name)

	mockClient.AssertExpectations(t)
}

// TestFetchRepositories_Error tests the fetchRepositories function for error case.
func TestFetchRepositories_Error(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("failed to fetch repositories"))

	repos, err := fetchRepositories("fake_token")
	assert.Error(t, err)
	assert.Nil(t, repos)
	assert.Contains(t, err.Error(), "failed to fetch repositories")
}

func TestFetchCommits(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"values": [
				{"hash": "commit1", "message": "Initial commit", "date": "2024-07-16T10:28:45.000+00:00",
					"author": {"user": {"display_name": "User1"}}},
				{"hash": "commit2", "message": "Update README", "date": "2024-07-17T11:35:22.000+00:00",
					"author": {"user": {"display_name": "User2"}}}
			]
		}`)),
	}, nil).Once()

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	commits, err := fetchCommits("fake_token", "repo1")
	assert.NoError(t, err)
	assert.Len(t, commits, 2)
	assert.Equal(t, "commit1", commits[0].Hash)
	assert.Equal(t, "commit2", commits[1].Hash)

	mockClient.AssertExpectations(t)
}

// TestFetchCommits_Error tests the fetchCommits function for error case.
func TestFetchCommits_Error(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("failed to fetch commits"))

	commits, err := fetchCommits("fake_token", "repo1")
	assert.Error(t, err)
	assert.Nil(t, commits)
	assert.Contains(t, err.Error(), "failed to fetch commits")
}

func TestFetchCommitDetails(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"message": "Initial commit",
			"hash": "commit1",
			"date": "2024-07-16T10:28:45.000+00:00",
			"author": {"user": {"display_name": "User1"}},
			"summary": {"lines_added": 10, "lines_deleted": 2}
		}`)),
	}, nil).Once()

	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"values": [
				{"type": "added", "path": {"to": "file1.txt"}},
				{"type": "modified", "path": {"to": "file2.txt"}},
				{"type": "removed", "path": {"to": "file3.txt"}}
			]
		}`)),
	}, nil).Once()

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	commitDetails, err := fetchCommitDetails("fake_token", "repo1", "commit1")
	assert.NoError(t, err)
	assert.Equal(t, "commit1", commitDetails.Hash)
	assert.Equal(t, 10, commitDetails.Summary.LinesAdded)
	assert.Equal(t, 2, commitDetails.Summary.LinesDeleted)
	assert.Len(t, commitDetails.Files, 3)

	mockClient.AssertExpectations(t)
}

// TestFetchCommitDetails_Error tests the fetchCommitDetails function for error case.
func TestFetchCommitDetails_Error(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("failed to fetch commit details"))

	commitDetails, err := fetchCommitDetails("fake_token", "repo1", "commit1")
	assert.Error(t, err)
	assert.Empty(t, commitDetails)
	assert.Contains(t, err.Error(), "failed to fetch commit details")
}

// func TestFetchCommitDetails_InvalidJSON(t *testing.T) {
// 	mockClient := new(MockHTTPClient)

// 	// Mock the first call to fetch commit details
// 	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
// 		return strings.Contains(req.URL.String(), "commit")
// 	})).Return(&http.Response{
// 		StatusCode: http.StatusOK,
// 		Body:       io.NopCloser(strings.NewReader(`{`)), // Invalid JSON
// 	}, nil).Once()

// 	// Mock the second call to fetch diffstat
// 	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
// 		return strings.Contains(req.URL.String(), "diffstat")
// 	})).Return(&http.Response{
// 		StatusCode: http.StatusOK,
// 		Body:       io.NopCloser(strings.NewReader(`{`)), // Invalid JSON for the second call
// 	}, nil).Once()

// 	oldHTTPClient := httpClient
// 	httpClient = mockClient
// 	defer func() { httpClient = oldHTTPClient }()

// 	_, err := fetchCommitDetails("fake_token", "repo1", "commit1")
// 	assert.Error(t, err)
// 	assert.True(t, strings.Contains(err.Error(), "unexpected end of JSON input") || strings.Contains(err.Error(), "unexpected EOF"))

// 	mockClient.AssertExpectations(t)
// }

// TestFetchCommitDetails_NonOKStatusCode tests the fetchCommitDetails function for non-OK status code.
func TestFetchCommitDetails_NonOKStatusCode(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil)

	commitDetails, err := fetchCommitDetails("fake_token", "repo1", "commit1")
	assert.Error(t, err)
	assert.Empty(t, commitDetails)
	assert.Contains(t, err.Error(), "failed to fetch commit details")
}

func TestFetchAndSaveCommits_Error(t *testing.T) {
	mockClient := new(MockHTTPClient)

	// Mock the response for fetching repositories with an error
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "repositories")
	})).Return(&http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader(`{"type": "error", "error": {"message": "Token is invalid or not supported for this endpoint."}}`)),
	}, nil).Once()

	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	err := FetchAndSaveCommits("invalid_token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch repositories")

	mockClient.AssertExpectations(t)
}

func TestFetchAndSaveCommits(t *testing.T) {
	mockClient := new(MockHTTPClient)

	// Mock the response for fetching repositories
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "repositories")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "values": [
                {"name": "repo1", "slug": "repo1", "project": {"name": "Project1"}},
                {"name": "repo2", "slug": "repo2", "project": {"name": "Project2"}}
            ]
        }`)),
	}, nil).Once()

	// Mock the response for fetching commits for repo1
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "repo1/commits")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "values": [
                {"hash": "commit1", "message": "Initial commit", "date": "2024-07-16T10:28:45.000+00:00",
                    "author": {"user": {"display_name": "User1"}}},
                {"hash": "commit2", "message": "Update README", "date": "2024-07-17T11:35:22.000+00:00",
                    "author": {"user": {"display_name": "User2"}}}
            ]
        }`)),
	}, nil).Once()

	// Mock the response for fetching commits for repo2
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "repo2/commits")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "values": [
                {"hash": "commit3", "message": "Initial commit", "date": "2024-07-16T10:28:45.000+00:00",
                    "author": {"user": {"display_name": "User3"}}},
                {"hash": "commit4", "message": "Update README", "date": "2024-07-17T11:35:22.000+00:00",
                    "author": {"user": {"display_name": "User4"}}}
            ]
        }`)),
	}, nil).Once()

	// Mock the response for fetching commit details for commit1
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "commit/commit1")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "message": "Initial commit",
            "hash": "commit1",
            "date": "2024-07-16T10:28:45.000+00:00",
            "author": {"user": {"display_name": "User1"}},
            "summary": {"lines_added": 10, "lines_deleted": 2}
        }`)),
	}, nil).Once()

	// Mock the response for fetching commit details for commit2
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "commit/commit2")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "message": "Update README",
            "hash": "commit2",
            "date": "2024-07-17T11:35:22.000+00:00",
            "author": {"user": {"display_name": "User2"}},
            "summary": {"lines_added": 15, "lines_deleted": 3}
        }`)),
	}, nil).Once()

	// Mock the response for fetching commit details for commit3
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "commit/commit3")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "message": "Initial commit",
            "hash": "commit3",
            "date": "2024-07-16T10:28:45.000+00:00",
            "author": {"user": {"display_name": "User3"}},
            "summary": {"lines_added": 5, "lines_deleted": 1}
        }`)),
	}, nil).Once()

	// Mock the response for fetching commit details for commit4
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "commit/commit4")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "message": "Update README",
            "hash": "commit4",
            "date": "2024-07-17T11:35:22.000+00:00",
            "author": {"user": {"display_name": "User4"}},
            "summary": {"lines_added": 20, "lines_deleted": 5}
        }`)),
	}, nil).Once()

	// Mock the response for fetching diffstat details for commit1
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "diffstat/commit1")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "values": [
                {"type": "added", "path": {"to": "file1.txt"}},
                {"type": "modified", "path": {"to": "file2.txt"}},
                {"type": "removed", "path": {"to": "file3.txt"}}
            ]
        }`)),
	}, nil).Once()

	// Mock the response for fetching diffstat details for commit2
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "diffstat/commit2")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "values": [
                {"type": "added", "path": {"to": "file4.txt"}},
                {"type": "modified", "path": {"to": "file5.txt"}},
                {"type": "removed", "path": {"to": "file6.txt"}}
            ]
        }`)),
	}, nil).Once()

	// Mock the response for fetching diffstat details for commit3
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "diffstat/commit3")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "values": [
                {"type": "added", "path": {"to": "file7.txt"}},
                {"type": "modified", "path": {"to": "file8.txt"}},
                {"type": "removed", "path": {"to": "file9.txt"}}
            ]
        }`)),
	}, nil).Once()

	// Mock the response for fetching diffstat details for commit4
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "diffstat/commit4")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
            "values": [
                {"type": "added", "path": {"to": "file10.txt"}},
                {"type": "modified", "path": {"to": "file11.txt"}},
                {"type": "removed", "path": {"to": "file12.txt"}}
            ]
        }`)),
	}, nil).Once()

	// Mock the database update operation
	mockCollection := new(MockCollection)
	mockCollection.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mongo.UpdateResult{}, nil).Times(4)

	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	oldGetCollection := db.GetCollectionFunc
	db.GetCollectionFunc = func() db.CollectionInterface {
		return mockCollection
	}
	defer func() { db.GetCollectionFunc = oldGetCollection }()

	err := FetchAndSaveCommits("fake_token")
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
	mockCollection.AssertExpectations(t)
}

func TestFetchCommitDetails_FailedToFetchDiffstat(t *testing.T) {
	mockClient := new(MockHTTPClient)

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "commit")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"hash": "commit1",
			"message": "Initial commit",
			"author": {"user": {"display_name": "User1"}},
			"date": "2024-07-16T10:28:45+00:00"
		}`)),
	}, nil).Once()

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "diffstat")
	})).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(`Internal Server Error`)),
	}, nil).Once()

	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	_, err := fetchCommitDetails("fake_token", "repo1", "commit1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch diffstat")

	mockClient.AssertExpectations(t)
}

// func TestFetchAndSaveCommits_FailedToLoadConfig(t *testing.T) {
// 	// Save the original loadConfigFunc
// 	originalLoadConfigFunc := loadConfigFunc

// 	// Mock the loadConfigFunc to return an error
// 	loadConfigFunc = func() (*config.Config, error) {
// 		return nil, fmt.Errorf("failed to load config")
// 	}
// 	defer func() {
// 		loadConfigFunc = originalLoadConfigFunc
// 	}()

// 	// This should result in the program calling log.Fatalf
// 	if os.Getenv("BE_CRASHER") == "1" {
// 		FetchAndSaveCommits("fake_token")
// 		return
// 	}

// 	cmd := exec.Command(os.Args[0], "-test.run=TestFetchAndSaveCommits_FailedToLoadConfig")
// 	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
// 	err := cmd.Run()

// 	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
// 		return
// 	}

// 	t.Fatalf("process ran with err %v, want exit status 1", err)
// }

// TestFetchRepositories_HTTPError simulates an HTTP error when fetching repositories.
func TestFetchRepositories_HTTPError(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("HTTP error"))

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	repos, err := fetchRepositories("fake_token")
	assert.Error(t, err)
	assert.Nil(t, repos)
	assert.Contains(t, err.Error(), "HTTP error")

	mockClient.AssertExpectations(t)
}

// TestFetchCommits_HTTPError simulates an HTTP error when fetching commits.
func TestFetchCommits_HTTPError(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("HTTP error"))

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	commits, err := fetchCommits("fake_token", "repo1")
	assert.Error(t, err)
	assert.Nil(t, commits)
	assert.Contains(t, err.Error(), "HTTP error")

	mockClient.AssertExpectations(t)
}

// TestFetchCommitDetails_HTTPError simulates an HTTP error when fetching commit details.
func TestFetchCommitDetails_HTTPError(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("HTTP error"))

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	commitDetails, err := fetchCommitDetails("fake_token", "repo1", "commit1")
	assert.Error(t, err)
	assert.Empty(t, commitDetails)
	assert.Contains(t, err.Error(), "HTTP error")

	mockClient.AssertExpectations(t)
}

// TestFetchAndSaveCommits_RepoFetchError simulates an error while fetching repositories.
func TestFetchAndSaveCommits_RepoFetchError(t *testing.T) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("HTTP error"))

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	err := FetchAndSaveCommits("fake_token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch repositories")

	mockClient.AssertExpectations(t)
}

// // TestFetchCommitDetails_InvalidJSON simulates invalid JSON response for commit details.
func TestFetchCommitDetails_InvalidJSON(t *testing.T) {
	mockClient := new(MockHTTPClient)

	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{`)), // Invalid JSON
	}, nil).Once()

	// Inject the mock client directly
	oldHTTPClient := httpClient
	httpClient = mockClient
	defer func() { httpClient = oldHTTPClient }()

	_, err := fetchCommitDetails("fake_token", "repo1", "commit1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected") // Check for a part of the error message

	mockClient.AssertExpectations(t)
}


// TestFetchAndSaveCommits_CommitFetchError simulates an error while fetching commits.
// func TestFetchAndSaveCommits_CommitFetchError(t *testing.T) {
// 	mockClient := new(MockHTTPClient)

// 	// Mock the response for fetching repositories
// 	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
// 		return strings.Contains(req.URL.String(), "repositories")
// 	})).Return(&http.Response{
// 		StatusCode: http.StatusOK,
// 		Body: io.NopCloser(strings.NewReader(`{
// 			"values": [
// 				{"name": "repo1", "slug": "repo1", "project": {"name": "Project1"}}
// 			]
// 		}`)),
// 	}, nil).Once()

// 	// Mock an error response for fetching commits
// 	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
// 		return strings.Contains(req.URL.String(), "repo1/commits")
// 	})).Return(nil, errors.New("HTTP error")).Once()

// 	// Inject the mock client directly
// 	oldHTTPClient := httpClient
// 	httpClient = mockClient
// 	defer func() { httpClient = oldHTTPClient }()

// 	// Call the function to test
// 	err := FetchAndSaveCommits("fake_token")

// 	// Check for the presence of an error
// 	assert.Error(t, err, "Expected an error due to failed commit fetch, but got nil")

// 	// Assert that the expectations were met
// 	mockClient.AssertExpectations(t)
// }