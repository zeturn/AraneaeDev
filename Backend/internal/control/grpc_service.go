package control

import (
	"context"
	"errors"
	"os"
	"strings"

	"araneae-go/gen/pb"
	"araneae-go/internal/common"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (a *App) GetArtifact(ctx context.Context, req *pb.GetArtifactRequest) (*pb.GetArtifactResponse, error) {
	node, ok := authenticatedNodeFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authenticated node")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing artifact metadata")
	}
	runID := strings.TrimSpace(firstMetadataValue(md, controlRunIDMetadata))
	runToken := strings.TrimSpace(firstMetadataValue(md, controlRunTokenMetadata))
	correlationID := strings.TrimSpace(firstMetadataValue(md, controlCorrelationIDMD))
	if runID == "" || runToken == "" || correlationID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing run authorization metadata")
	}

	var run common.TaskRun
	if err := a.db.WithContext(ctx).Where("id = ?", runID).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.PermissionDenied, "run not found")
		}
		return nil, status.Error(codes.Internal, "load run failed")
	}
	if run.CorrelationID != correlationID {
		return nil, status.Error(codes.PermissionDenied, "run correlation mismatch")
	}
	if hashNodeKey(runToken) != run.RunTokenHash {
		return nil, status.Error(codes.PermissionDenied, "run token mismatch")
	}
	if strings.TrimSpace(node.CeleryQueue) != strings.TrimSpace(run.NodeQueue) {
		return nil, status.Error(codes.PermissionDenied, "node queue mismatch")
	}
	if isTerminalRunStatus(run.Status) {
		return nil, status.Error(codes.PermissionDenied, "run already finished")
	}

	var v common.ArtifactVersion
	if err := a.db.WithContext(ctx).Where("id = ? AND project_id = ?", req.VersionId, req.ProjectId).First(&v).Error; err != nil {
		return nil, err
	}
	content, err := os.ReadFile(v.StoragePath)
	if err != nil {
		return nil, err
	}
	return &pb.GetArtifactResponse{
		FileName: v.FileName,
		Content:  content,
		Sha256:   v.SHA256,
	}, nil
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
