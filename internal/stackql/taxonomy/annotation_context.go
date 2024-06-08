package taxonomy

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
)

// TODO:
//   - For views, need API to get child.
type AnnotationCtx interface {
	GetHIDs() internaldto.HeirarchyIdentifiers
	IsDynamic() bool
	GetView() (internaldto.RelationDTO, bool)
	GetSubquery() (internaldto.SubqueryDTO, bool)
	GetInputTableName() (string, error)
	GetParameters() map[string]interface{}
	GetSchema() anysdk.Schema
	GetTableMeta() tablemetadata.ExtendedTableMetadata
	Prepare(handlerCtx handler.HandlerContext, inStream streaming.MapStream) error
	SetDynamic()
	Clone() AnnotationCtx
}

type standardAnnotationCtx struct {
	isDynamic  bool
	schema     anysdk.Schema
	hIDs       internaldto.HeirarchyIdentifiers
	tableMeta  tablemetadata.ExtendedTableMetadata
	parameters map[string]interface{}
}

func NewStaticStandardAnnotationCtx(
	schema anysdk.Schema,
	hIds internaldto.HeirarchyIdentifiers,
	tableMeta tablemetadata.ExtendedTableMetadata,
	parameters map[string]interface{},
) AnnotationCtx {
	return &standardAnnotationCtx{
		isDynamic:  false,
		schema:     schema,
		hIDs:       hIds,
		tableMeta:  tableMeta,
		parameters: parameters,
	}
}

func (ac *standardAnnotationCtx) Clone() AnnotationCtx {
	clonedParams := make(map[string]interface{})
	for k, v := range ac.parameters {
		clonedParams[k] = v
	}
	return &standardAnnotationCtx{
		isDynamic:  ac.isDynamic,
		schema:     ac.schema,
		hIDs:       ac.hIDs,
		tableMeta:  ac.tableMeta,
		parameters: clonedParams,
	}
}

func (ac *standardAnnotationCtx) IsDynamic() bool {
	return ac.isDynamic
}

func (ac *standardAnnotationCtx) GetView() (internaldto.RelationDTO, bool) {
	return ac.hIDs.GetView()
}

func (ac *standardAnnotationCtx) GetSubquery() (internaldto.SubqueryDTO, bool) {
	return ac.hIDs.GetSubquery()
}

func (ac *standardAnnotationCtx) SetDynamic() {
	ac.isDynamic = true
}

func (ac *standardAnnotationCtx) Prepare(
	handlerCtx handler.HandlerContext, //nolint:revive // future proofing
	stream streaming.MapStream,
) error {
	// TODO: accomodate SQL data source
	sqlDataSource, isSQLDataSource := ac.GetTableMeta().GetSQLDataSource()
	if isSQLDataSource {
		ac.tableMeta.SetSQLDataSource(sqlDataSource)
		// TODO: persist mirror table here a la GenerateInsertDML()
		// anTab := util.NewAnnotatedTabulation(tab, ac.GetHIDs(), inputTableName, annotationCtx.GetTableMeta().GetAlias())
		// ddl, err := handlerCtx.GetDrmConfig().GenerateDDL(ac.tableMeta, nil, 0, false)
		return nil
	}
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
	params := ac.GetParameters()
	// LAZY EVAL if dynamic
	if ac.isDynamic {
		viewDTO, isView := ac.GetView()
		// TODO: fill this out
		if isView {
			logging.GetLogger().Debugf("viewDTO = %v\n", viewDTO)
		}
		prov, provErr := pr.GetProvider()
		if provErr != nil {
			return provErr
		}
		ac.tableMeta.WithGetHTTPArmoury(
			func() (anysdk.HTTPArmoury, error) {
				httpPreparator := anysdk.NewHTTPPreparator(
					prov,
					svc,
					opStore,
					nil,
					stream,
					nil,
					logging.GetLogger(),
				)
				httpArmoury, armouryErr := httpPreparator.BuildHTTPRequestCtxFromAnnotation()
				return httpArmoury, armouryErr
			},
		)
		return nil
	}

	ac.tableMeta.WithGetHTTPArmoury(
		func() (anysdk.HTTPArmoury, error) {
			// need to dynamically generate stream, otherwise repeated calls result in empty body
			parametersCleaned, cleanErr := util.TransformSQLRawParameters(params)
			if cleanErr != nil {
				return nil, cleanErr
			}
			stream.Write( //nolint:errcheck // TODO: handle error
				[]map[string]interface{}{
					parametersCleaned,
				},
			)
			prov, provErr := pr.GetProvider()
			if provErr != nil {
				return nil, provErr
			}
			httpPreparator := anysdk.NewHTTPPreparator(
				prov,
				svc,
				opStore,
				nil,
				stream,
				nil,
				logging.GetLogger(),
			)
			httpArmoury, armouryErr := httpPreparator.BuildHTTPRequestCtxFromAnnotation()
			if armouryErr != nil {
				return nil, armouryErr
			}
			return httpArmoury, nil
		},
	)
	return nil
}

func (ac *standardAnnotationCtx) GetHIDs() internaldto.HeirarchyIdentifiers {
	return ac.hIDs
}

func (ac *standardAnnotationCtx) GetParameters() map[string]interface{} {
	return ac.parameters
}

func (ac *standardAnnotationCtx) GetSchema() anysdk.Schema {
	return ac.schema
}

func (ac *standardAnnotationCtx) GetInputTableName() (string, error) {
	return ac.tableMeta.GetInputTableName()
}

func (ac *standardAnnotationCtx) GetTableMeta() tablemetadata.ExtendedTableMetadata {
	return ac.tableMeta
}
