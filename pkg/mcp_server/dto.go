package mcp_server //nolint:revive,stylecheck // fine for now

type GreetingInput struct {
	Name string `json:"name" jsonschema:"the name of the person to greet"`
}

type GreetingOutput struct {
	Greeting string `json:"greeting" jsonschema:"the greeting to tell to the user"`
}

/*

Comment AA

Please turn the below python classes into golang structures of the same name with json and yaml attributes exposed

```python
class QueryInput(BaseModel):
    sql: str = Field(description="SQL statement to execute")
    parameters: Optional[List[Any]] = Field(default=None, description="Positional parameters for the SQL")
    row_limit: int = Field(default=500, ge=1, le=10000, description="Max rows to return for SELECT queries")
    format: Literal["markdown", "json"] = Field(default="markdown", description="Output format for results")


class QueryJSONInput(BaseModel):
    sql: str
    parameters: Optional[List[Any]] = None
    row_limit: int = 500

class ListSchemasInput(BaseModel):
    include_system: bool = Field(default=False, description="Include pg_* and information_schema")
    include_temp: bool = Field(default=False, description="Include temporary schemas (pg_temp_*)")
    require_usage: bool = Field(default=True, description="Only list schemas with USAGE privilege")
    row_limit: int = Field(default=10000, ge=1, le=100000, description="Maximum number of schemas to return")
    name_like: Optional[str] = Field(default=None, description="Filter schema names by LIKE pattern (use % and _). '*' and '?' will be translated.")
    case_sensitive: bool = Field(default=False, description="When true, use LIKE instead of ILIKE for name_like")

class ListSchemasPageInput(BaseModel):
    include_system: bool = False
    include_temp: bool = False
    require_usage: bool = True
    page_size: int = Field(default=500, ge=1, le=10000)
    cursor: Optional[str] = None
    name_like: Optional[str] = None
    case_sensitive: bool = False


class ListTablesInput(BaseModel):
    db_schema: Optional[str] = Field(default=None, description="Schema to list tables from; defaults to current_schema()")
    name_like: Optional[str] = Field(default=None, description="Filter table_name by pattern; '*' and '?' translate to SQL wildcards")
    case_sensitive: bool = Field(default=False, description="Use LIKE (true) or ILIKE (false) for name_like")
    table_types: Optional[List[str]] = Field(
        default=None,
        description="Limit to specific information_schema table_type values (e.g., 'BASE TABLE','VIEW')",
    )
    row_limit: int = Field(default=10000, ge=1, le=100000)


class ListTablesPageInput(BaseModel):
    db_schema: Optional[str] = None
    name_like: Optional[str] = None
    case_sensitive: bool = False
    table_types: Optional[List[str]] = None
    page_size: int = Field(default=500, ge=1, le=10000)
    cursor: Optional[str] = None

    def to_list_tables_input(self) -> ListTablesInput:
        return ListTablesInput(
            db_schema=self.db_schema,
            name_like=self.name_like,
            case_sensitive=self.case_sensitive,
            table_types=self.table_types,
            row_limit=self.page_size
        )
```

*/

type GreetInput struct {
	Name string `json:"name" jsonschema:"the person to greet"`
}

type HierarchyInput struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Service  string `json:"service,omitempty" yaml:"service,omitempty"`
	Resource string `json:"resource,omitempty" yaml:"resource,omitempty"`
	Method   string `json:"method,omitempty" yaml:"method,omitempty"`
	RowLimit int    `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
	Format   string `json:"format,omitempty" yaml:"format,omitempty"`
	// Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type ServerInfoOutput struct {
	Name       string `json:"name" jsonschema:"server name"`
	Info       string `json:"info" jsonschema:"server info"`
	IsReadOnly bool   `json:"read_only" jsonschema:"is the database read-only"`
}

type QueryInput struct {
	SQL      string `json:"sql" yaml:"sql"`
	RowLimit int    `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
	Format   string `json:"format,omitempty" yaml:"format,omitempty"`
	// Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type QueryJSONInput struct {
	SQL      string `json:"sql" yaml:"sql"`
	RowLimit int    `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
	// Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type ListSchemasInput struct {
	IncludeSystem bool    `json:"include_system,omitempty" yaml:"include_system,omitempty"`
	IncludeTemp   bool    `json:"include_temp,omitempty" yaml:"include_temp,omitempty"`
	RequireUsage  bool    `json:"require_usage,omitempty" yaml:"require_usage,omitempty"`
	RowLimit      int     `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
	NameLike      *string `json:"name_like,omitempty" yaml:"name_like,omitempty"`
	CaseSensitive bool    `json:"case_sensitive,omitempty" yaml:"case_sensitive,omitempty"`
	Format        string  `json:"format,omitempty" yaml:"format,omitempty"`
}

type ListSchemasPageInput struct {
	IncludeSystem bool    `json:"include_system" yaml:"include_system"`
	IncludeTemp   bool    `json:"include_temp" yaml:"include_temp"`
	RequireUsage  bool    `json:"require_usage" yaml:"require_usage"`
	PageSize      int     `json:"page_size" yaml:"page_size"`
	Cursor        *string `json:"cursor,omitempty" yaml:"cursor,omitempty"`
	NameLike      *string `json:"name_like,omitempty" yaml:"name_like,omitempty"`
	CaseSensitive bool    `json:"case_sensitive" yaml:"case_sensitive"`
	Format        string  `json:"format" yaml:"format"`
}

type ListTablesInput struct {
	Hierarchy     *HierarchyInput `json:"hierarchy,omitempty" yaml:"hierarchy,omitempty"`
	NameLike      *string         `json:"name_like,omitempty" yaml:"name_like,omitempty"`
	CaseSensitive bool            `json:"case_sensitive,omitempty" yaml:"case_sensitive,omitempty"`
	TableTypes    []string        `json:"table_types,omitempty" yaml:"table_types,omitempty"`
	RowLimit      int             `json:"row_limit,omitempty" yaml:"row_limit,omitempty"`
	Format        string          `json:"format,omitempty" yaml:"format,omitempty"`
}

type ListTablesPageInput struct {
	Hierarchy     *HierarchyInput `json:"hierarchy,omitempty" yaml:"hierarchy,omitempty"`
	NameLike      *string         `json:"name_like,omitempty" yaml:"name_like,omitempty"`
	CaseSensitive bool            `json:"case_sensitive,omitempty" yaml:"case_sensitive,omitempty"`
	TableTypes    []string        `json:"table_types,omitempty" yaml:"table_types,omitempty"`
	PageSize      int             `json:"page_size,omitempty" yaml:"page_size,omitempty"`
	Cursor        *string         `json:"cursor,omitempty" yaml:"cursor,omitempty"`
	Format        string          `json:"format,omitempty" yaml:"format,omitempty"`
}
