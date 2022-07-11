package taxonomy

import (
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/util"
)

type AnnotationCtx interface {
	GetHIDs() *dto.HeirarchyIdentifiers
	IsDynamic() bool
	GetParameters() map[string]interface{}
	GetSchema() *openapistackql.Schema
	GetTableMeta() *ExtendedTableMetadata
	Prepare(handlerCtx *handler.HandlerContext, inStream streaming.MapStream) error
	SetDynamic()
}

type StandardAnnotationCtx struct {
	isDynamic  bool
	Schema     *openapistackql.Schema
	HIDs       *dto.HeirarchyIdentifiers
	TableMeta  *ExtendedTableMetadata
	Parameters map[string]interface{}
}

func NewStaticStandardAnnotationCtx(
	schema *openapistackql.Schema,
	hIds *dto.HeirarchyIdentifiers,
	tableMeta *ExtendedTableMetadata,
	parameters map[string]interface{},
) AnnotationCtx {
	return &StandardAnnotationCtx{
		isDynamic:  false,
		Schema:     schema,
		HIDs:       hIds,
		TableMeta:  tableMeta,
		Parameters: parameters,
	}
}

func (ac *StandardAnnotationCtx) IsDynamic() bool {
	return ac.isDynamic
}

func (ac *StandardAnnotationCtx) SetDynamic() {
	ac.isDynamic = true
}

func (ac *StandardAnnotationCtx) Prepare(
	handlerCtx *handler.HandlerContext,
	stream streaming.MapStream,
) error {
	pr, err := ac.GetTableMeta().GetProvider()
	if err != nil {
		return err
	}
	svc, err := ac.GetTableMeta().GetService()
	if err != nil {
		return err
	}
	opStore, err := ac.GetTableMeta().GetMethod()
	if err != nil {
		return err
	}
	if ac.isDynamic {
		// LAZY EVAL
		ac.TableMeta.GetHttpArmoury = func() (httpbuild.HTTPArmoury, error) {
			httpArmoury, err := httpbuild.BuildHTTPRequestCtxFromAnnotation(handlerCtx, stream, pr, opStore, svc, nil, nil)
			return httpArmoury, err
		}
		return nil
	} else {
		parametersCleaned, err := util.TransformSQLRawParameters(ac.GetParameters())
		if err != nil {
			return err
		}
		stream.Write(
			[]map[string]interface{}{
				parametersCleaned,
			},
		)
	}
	httpArmoury, err := httpbuild.BuildHTTPRequestCtxFromAnnotation(handlerCtx, stream, pr, opStore, svc, nil, nil)
	if err != nil {
		return err
	}
	ac.TableMeta.GetHttpArmoury = func() (httpbuild.HTTPArmoury, error) { return httpArmoury, nil }
	return nil
}

func (ac *StandardAnnotationCtx) GetHIDs() *dto.HeirarchyIdentifiers {
	return ac.HIDs
}

func (ac *StandardAnnotationCtx) GetParameters() map[string]interface{} {
	return ac.Parameters
}

func (ac *StandardAnnotationCtx) GetSchema() *openapistackql.Schema {
	return ac.Schema
}

func (ac *StandardAnnotationCtx) GetTableMeta() *ExtendedTableMetadata {
	return ac.TableMeta
}
