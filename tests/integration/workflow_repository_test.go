package integration

import (
	"testing"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.DB.DB)

	// Create fixture
	fixtures := testutil.NewFixtureBuilder()
	workflowFixture := fixtures.Workflow()

	// Convert to CreateWorkflowRequest
	req := &models.CreateWorkflowRequest{
		WorkflowID:  workflowFixture.WorkflowID,
		Version:     workflowFixture.Version,
		Name:        workflowFixture.Name,
		Description: workflowFixture.Description,
		Definition:  workflowFixture.Definition,
		Tags:        workflowFixture.Tags,
	}

	// Test Create
	createdBy := uuid.New()
	workflow, err := repo.Create(ctx, req, &createdBy)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, workflow.ID)
	assert.Equal(t, req.WorkflowID, workflow.WorkflowID)
	assert.Equal(t, req.Name, workflow.Name)
	assert.Equal(t, req.Version, workflow.Version)

	// Verify it was created
	retrieved, err := repo.GetByID(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Equal(t, workflow.WorkflowID, retrieved.WorkflowID)
	assert.Equal(t, workflow.Name, retrieved.Name)
	assert.Equal(t, workflow.Version, retrieved.Version)
}

func TestWorkflowRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.DB.DB)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflowFixture := fixtures.Workflow()

	req := &models.CreateWorkflowRequest{
		WorkflowID:  workflowFixture.WorkflowID,
		Version:     workflowFixture.Version,
		Name:        workflowFixture.Name,
		Description: workflowFixture.Description,
		Definition:  workflowFixture.Definition,
		Tags:        workflowFixture.Tags,
	}

	createdBy := uuid.New()
	workflow, err := repo.Create(ctx, req, &createdBy)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, retrieved.ID)
	assert.Equal(t, workflow.WorkflowID, retrieved.WorkflowID)
}

func TestWorkflowRepository_GetByWorkflowID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.DB.DB)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflowFixture := fixtures.Workflow()

	req := &models.CreateWorkflowRequest{
		WorkflowID:  "test-workflow-123",
		Version:     "1.0.0",
		Name:        workflowFixture.Name,
		Description: workflowFixture.Description,
		Definition:  workflowFixture.Definition,
		Tags:        workflowFixture.Tags,
	}

	createdBy := uuid.New()
	workflow, err := repo.Create(ctx, req, &createdBy)
	require.NoError(t, err)

	// Test GetByWorkflowID
	retrieved, err := repo.GetByWorkflowID(ctx, "test-workflow-123")
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, retrieved.ID)
	assert.Equal(t, "test-workflow-123", retrieved.WorkflowID)
	assert.Equal(t, "1.0.0", retrieved.Version)
}

func TestWorkflowRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.DB.DB)

	// Create multiple workflows
	fixtures := testutil.NewFixtureBuilder()
	createdBy := uuid.New()

	for i := 0; i < 5; i++ {
		workflowFixture := fixtures.Workflow()
		req := &models.CreateWorkflowRequest{
			WorkflowID:  uuid.New().String(),
			Version:     workflowFixture.Version,
			Name:        workflowFixture.Name,
			Description: workflowFixture.Description,
			Definition:  workflowFixture.Definition,
			Tags:        workflowFixture.Tags,
		}
		_, err := repo.Create(ctx, req, &createdBy)
		require.NoError(t, err)
	}

	// Test List
	workflows, total, err := repo.List(ctx, nil, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(workflows), 5)
	assert.GreaterOrEqual(t, total, int64(5))
}

func TestWorkflowRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.DB.DB)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflowFixture := fixtures.Workflow()

	createReq := &models.CreateWorkflowRequest{
		WorkflowID:  workflowFixture.WorkflowID,
		Version:     workflowFixture.Version,
		Name:        workflowFixture.Name,
		Description: workflowFixture.Description,
		Definition:  workflowFixture.Definition,
		Tags:        workflowFixture.Tags,
	}

	createdBy := uuid.New()
	workflow, err := repo.Create(ctx, createReq, &createdBy)
	require.NoError(t, err)

	// Update workflow
	updatedName := "Updated Workflow Name"
	updatedDescription := "Updated description"
	updateReq := &models.UpdateWorkflowRequest{
		Name:        &updatedName,
		Description: &updatedDescription,
	}

	updated, err := repo.Update(ctx, workflow.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Workflow Name", updated.Name)

	// Verify update
	retrieved, err := repo.GetByID(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Workflow Name", retrieved.Name)
	assert.Equal(t, "Updated description", *retrieved.Description)
}

func TestWorkflowRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.DB.DB)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflowFixture := fixtures.Workflow()

	req := &models.CreateWorkflowRequest{
		WorkflowID:  workflowFixture.WorkflowID,
		Version:     workflowFixture.Version,
		Name:        workflowFixture.Name,
		Description: workflowFixture.Description,
		Definition:  workflowFixture.Definition,
		Tags:        workflowFixture.Tags,
	}

	createdBy := uuid.New()
	workflow, err := repo.Create(ctx, req, &createdBy)
	require.NoError(t, err)

	// Delete workflow
	err = repo.Delete(ctx, workflow.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, workflow.ID)
	assert.Error(t, err)
}
