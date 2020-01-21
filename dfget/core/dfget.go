package core

import (
	"context"
	"fmt"
	"io"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/downloader"
	backDown "github.com/dragonflyoss/Dragonfly/dfget/core/downloader/back_downloader"
	p2pDown "github.com/dragonflyoss/Dragonfly/dfget/core/downloader/p2p_downloader"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/printer"
)

type dfGet struct {
}

type DFGet interface {
	GetFile(ctx context.Context, cfg *config.Config) error
	GetReader(ctx context.Context, cfg *config.Config) (io.Reader, error)
}

func NewDFGet() DFGet {
	return &dfGet{}
}

func (df *dfGet) GetFile(ctx context.Context, cfg *config.Config) error {
	return Start(cfg)
}

func (df *dfGet) GetReader(ctx context.Context, cfg *config.Config) (io.Reader, error) {
	cfg.RV.StreamMode = true

	var (
		supernodeAPI = api.NewSupernodeAPI()
		register     = regist.NewSupernodeRegister(cfg, supernodeAPI)
		err          error
		result       *regist.RegisterResult
	)

	printer.Println(fmt.Sprintf("--%s--  %s",
		cfg.StartTime.Format(config.DefaultTimestampFormat), cfg.URL))

	if err = prepareStream(cfg); err != nil {
		return nil, errortypes.New(config.CodePrepareError, err.Error())
	}

	if result, err = registerToSuperNode(cfg, register); err != nil {
		return nil, errortypes.New(config.CodeRegisterError, err.Error())
	}

	r, err := df.getReader(ctx, cfg, supernodeAPI, register, result)
	if err != nil {
		return nil, errortypes.New(config.CodeDownloadError, err.Error())
	}

	return r, nil
}

func (df *dfGet) getReader(ctx context.Context, cfg *config.Config, supernodeAPI api.SupernodeAPI,
	register regist.SupernodeRegister, result *regist.RegisterResult) (io.Reader, error) {
	timeout := calculateTimeout(cfg)
	var getter downloader.Downloader

	if cfg.BackSourceReason > 0 {
		getter = backDown.NewBackDownloader(cfg, result)
	} else {
		printer.Printf("start download by dragonfly...")
		getter = p2pDown.NewP2PDownloader(cfg, supernodeAPI, register, result)
	}
	ctx, _ = context.WithTimeout(ctx, timeout)
	return getter.RunStream(ctx)
}
