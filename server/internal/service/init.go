package service

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

type WorkspaceService struct{}

func NewWorkspaceService() *WorkspaceService {
	return &WorkspaceService{}
}

type TableSchemaService struct{}

func NewTableSchemaService() *TableSchemaService {
	return &TableSchemaService{}
}

type QueryService struct{}

func NewQueryService() *QueryService {
	return &QueryService{}
}
