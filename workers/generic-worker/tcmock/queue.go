package tcmock

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/taskcluster/taskcluster/v29/clients/client-go/tcqueue"
	"github.com/taskcluster/taskcluster/v29/workers/generic-worker/s3"
)

type Queue struct {
	t         *testing.T
	publisher s3.Publisher

	// key: "<taskId>"
	tasks map[string]*tcqueue.TaskDefinitionAndStatus

	// key: "<taskId>:<runId>:<name>"
	artifacts map[string]*tcqueue.Artifact

	// key: "<taskId>:<runId>:<name>"
	artifactContent map[string][]byte
}

/////////////////////////////////////////////////

func (queue *Queue) ClaimWork(provisionerId, workerType string, payload *tcqueue.ClaimWorkRequest) (*tcqueue.ClaimWorkResponse, error) {
	maxTasks := payload.Tasks
	tasks := []tcqueue.TaskClaim{}
	for _, j := range queue.tasks {
		if j.Task.WorkerType == workerType && j.Task.ProvisionerID == provisionerId && j.Status.State == "pending" {
			j.Status.State = "running"
			j.Status.Runs = []tcqueue.RunInformation{
				{
					RunID:         0,
					ReasonCreated: "scheduled",
				},
			}
			tasks = append(
				tasks,
				tcqueue.TaskClaim{
					Task:   j.Task,
					Status: j.Status,
				},
			)
			if len(tasks) == int(maxTasks) {
				break
			}
		}
	}
	return &tcqueue.ClaimWorkResponse{
		Tasks: tasks,
	}, nil
}

func (queue *Queue) CreateArtifact(taskId, runId, name string, payload *tcqueue.PostArtifactRequest) (*tcqueue.PostArtifactResponse, error) {
	var request tcqueue.Artifact
	err := json.Unmarshal([]byte(*payload), &request)
	if err != nil {
		queue.t.Fatalf("Error unmarshalling from json: %v", err)
	}
	queue.artifacts[taskId+":"+runId+":"+name] = &request
	var response interface{}
	switch request.StorageType {
	case "s3":
		var s3Request tcqueue.S3ArtifactRequest
		err := json.Unmarshal([]byte(*payload), &s3Request)
		if err != nil {
			queue.t.Fatalf("Error unmarshalling S3 Artifact Request from json: %v", err)
		}
		response = tcqueue.S3ArtifactResponse{
			ContentType: s3Request.ContentType,
			Expires:     s3Request.Expires,
			PutURL:      "http://localhost:12453",
			StorageType: s3Request.StorageType,
		}
	case "error":
		var errorRequest tcqueue.ErrorArtifactRequest
		err := json.Unmarshal([]byte(*payload), &errorRequest)
		if err != nil {
			queue.t.Fatalf("Error unmarshalling Error Artifact Request from json: %v", err)
		}
		response = tcqueue.ErrorArtifactResponse{
			StorageType: errorRequest.StorageType,
		}
	case "reference":
		var redirectRequest tcqueue.RedirectArtifactRequest
		err := json.Unmarshal([]byte(*payload), &redirectRequest)
		if err != nil {
			queue.t.Fatalf("Error unmarshalling Redirect Artifact Request from json: %v", err)
		}
		response = tcqueue.RedirectArtifactResponse{
			StorageType: redirectRequest.StorageType,
		}
	default:
		queue.t.Fatalf("Unrecognised storage type: %v", request.StorageType)
	}
	var par tcqueue.PostArtifactResponse
	par, err = json.Marshal(response)
	if err != nil {
		queue.t.Fatalf("Error marshalling into json: %v", err)
	}
	return &par, nil
}

func (queue *Queue) CreateTask(taskId string, payload *tcqueue.TaskDefinitionRequest) (*tcqueue.TaskStatusResponse, error) {
	queue.tasks[taskId] = &tcqueue.TaskDefinitionAndStatus{
		Status: tcqueue.TaskStatusStructure{
			TaskID: taskId,
			State:  "pending",
		},
		Task: tcqueue.TaskDefinitionResponse{
			Created:       payload.Created,
			Deadline:      payload.Deadline,
			Dependencies:  payload.Dependencies,
			Expires:       payload.Expires,
			Extra:         payload.Extra,
			Metadata:      payload.Metadata,
			Payload:       payload.Payload,
			Priority:      payload.Priority,
			ProvisionerID: payload.ProvisionerID,
			Requires:      payload.Requires,
			Retries:       payload.Retries,
			Routes:        payload.Routes,
			SchedulerID:   payload.SchedulerID,
			Scopes:        payload.Scopes,
			Tags:          payload.Tags,
			TaskGroupID:   payload.TaskGroupID,
			WorkerType:    payload.WorkerType,
		},
	}
	tsr := &tcqueue.TaskStatusResponse{
		Status: tcqueue.TaskStatusStructure{
			Deadline:      payload.Deadline,
			Expires:       payload.Expires,
			ProvisionerID: payload.ProvisionerID,
			RetriesLeft:   payload.Retries,
			Runs:          []tcqueue.RunInformation{},
			SchedulerID:   payload.SchedulerID,
			State:         "pending",
			TaskGroupID:   payload.TaskGroupID,
			TaskID:        taskId,
			WorkerType:    payload.WorkerType,
		},
	}
	return tsr, nil
}

func (queue *Queue) GetLatestArtifact_SignedURL(taskId, name string, duration time.Duration) (*url.URL, error) {
	return &url.URL{}, nil
}

func (queue *Queue) ListArtifacts(taskId, runId, continuationToken, limit string) (*tcqueue.ListArtifactsResponse, error) {
	return &tcqueue.ListArtifactsResponse{}, nil
}

func (queue *Queue) ReclaimTask(taskId, runId string) (*tcqueue.TaskReclaimResponse, error) {
	return &tcqueue.TaskReclaimResponse{}, nil
}

func (queue *Queue) ReportCompleted(taskId, runId string) (*tcqueue.TaskStatusResponse, error) {
	queue.tasks[taskId].Status.Runs[0].ReasonResolved = "completed"
	queue.tasks[taskId].Status.Runs[0].State = "completed"
	return queue.Status(taskId)
}

func (queue *Queue) ReportException(taskId, runId string, payload *tcqueue.TaskExceptionRequest) (*tcqueue.TaskStatusResponse, error) {
	queue.tasks[taskId].Status.Runs[0].ReasonResolved = payload.Reason
	queue.tasks[taskId].Status.Runs[0].State = "exception"
	return queue.Status(taskId)
}

func (queue *Queue) ReportFailed(taskId, runId string) (*tcqueue.TaskStatusResponse, error) {
	queue.tasks[taskId].Status.Runs[0].ReasonResolved = "failed"
	queue.tasks[taskId].Status.Runs[0].State = "failed"
	return queue.Status(taskId)
}

func (queue *Queue) Status(taskId string) (*tcqueue.TaskStatusResponse, error) {
	return &tcqueue.TaskStatusResponse{
		Status: queue.tasks[taskId].Status,
	}, nil
}

func (queue *Queue) Task(taskId string) (*tcqueue.TaskDefinitionResponse, error) {
	return &queue.tasks[taskId].Task, nil
}

/////////////////////////////////////////////////

func NewQueue(t *testing.T, publisher s3.Publisher) *Queue {
	return &Queue{
		t:         t,
		tasks:     map[string]*tcqueue.TaskDefinitionAndStatus{},
		artifacts: map[string]*tcqueue.Artifact{},
		publisher: publisher,
	}
}
