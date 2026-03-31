package control

import (
	"context"
	"os"

	"araneae-go/gen/pb"
	"araneae-go/internal/common"
)

func (a *App) GetArtifact(ctx context.Context, req *pb.GetArtifactRequest) (*pb.GetArtifactResponse, error) {
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
