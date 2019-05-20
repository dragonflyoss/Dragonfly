package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// TaskMgr as an interface defines all operations against Task.
// A Task will store some meta info about the taskFile, pieces and something else.
// A Task has a one-to-one correspondence with a file on the disk which is identified by taskID.
type TaskMgr interface {
	// Register a task represents that someone wants to download a file.
	// Supernode will get the task file meta and return taskID.
	// NOTE: If supernode cannot find the task file, the CDN download will be triggered.
	Register(ctx context.Context, task *types.TaskInfo) (taskID string, err error)

	// Get the task Info with specified taskID.
	Get(ctx context.Context, taskID string) (*types.TaskInfo, error)

	// List returns the list tasks with filter.
	List(ctx context.Context, filter map[string]string) ([]*types.TaskInfo, error)

	// CheckTaskStatus check whether the taskID corresponding file exists.
	CheckTaskStatus(ctx context.Context, taskID string) (bool, error)

	// DeleteTask delete a task
	// NOTE: delete the related peers and dfgetTask info is necessary.
	DeleteTask(ctx context.Context, taskID string) error

	// update the task info with specified info.
	// In common, there are several situations that we will use this method:
	// 1. when finished to download, update task status.
	// 2. for operation usage.
	UpdateTaskInfo(ctx context.Context, taskID string, taskInfo *types.TaskInfo) error

	// GetPieces get the pieces to be downloaded based on the scheduling result,
	// just like this: which pieces can be downloaded from which peers.
	GetPieces(ctx context.Context, taskID, clientID string, piecePullRequest *types.PiecePullRequest) ([]*types.PieceInfo, error)

	// UpdatePieceStatus update the piece status with specified parameters.
	// A task file is divided into several pieces logically.
	// We use a sting called pieceRange to identify a piece.
	// A pieceRange separated by a dash, like this: 0-45565, etc.
	UpdatePieceStatus(ctx context.Context, taskID, pieceRange string, pieceUpdateRequest *types.PieceUpdateRequest) error

	// GetPieceMD5 returns the md5 of pieceNum for taskID.
	GetPieceMD5(ctx context.Context, taskID string, pieceNum int) (pieceMD5 string, err error)

	// SetPieceMD5 set the md5 for pieceNum of taskID.
	SetPieceMD5(ctx context.Context, taskID string, pieceNum int, pieceMD5 string) (err error)

	// GetPieceMD5sByTaskID returns all pieceMD5s as a string slice.
	// All pieceMD5s are returned only if the CDN status is successful.
	GetPieceMD5sByTaskID(ctx context.Context, taskID string) (pieceMD5s []string, err error)
}
