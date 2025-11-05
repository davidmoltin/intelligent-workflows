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
	repo := postgres.NewWorkflowRepository(suite.Pool)

	// Create fixture
	fixtures := testutil.NewFixtureBuilder()
	workflow := fixtures.Workflow()

	// Test Create
	err := repo.Create(ctx, workflow)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, workflow.ID)

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
	repo := postgres.NewWorkflowRepository(suite.Pool)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflow := fixtures.Workflow()
	err := repo.Create(ctx, workflow)
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
	repo := postgres.NewWorkflowRepository(suite.Pool)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflow := fixtures.Workflow(func(w *models.Workflow) {
		w.WorkflowID = "test-workflow-123"
		w.Version = "1.0.0"
	})
	err := repo.Create(ctx, workflow)
	require.NoError(t, err)

	// Test GetByWorkflowID
	retrieved, err := repo.GetByWorkflowID(ctx, "test-workflow-123", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, retrieved.ID)
	assert.Equal(t, "test-workflow-123", retrieved.WorkflowID)
}

func TestWorkflowRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.Pool)

	// Create multiple workflows
	fixtures := testutil.NewFixtureBuilder()
	for i := 0; i < 5; i++ {
		workflow := fixtures.Workflow(func(w *models.Workflow) {
			w.WorkflowID = uuid.New().String()
		})
		err := repo.Create(ctx, workflow)
		require.NoError(t, err)
	}

	// Test List
	workflows, err := repo.List(ctx, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(workflows), 5)
}

func TestWorkflowRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	ctx := suite.GetContext(t)
	repo := postgres.NewWorkflowRepository(suite.Pool)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflow := fixtures.Workflow()
	err := repo.Create(ctx, workflow)
	require.NoError(t, err)

	// Update workflow
	workflow.Name = "Updated Workflow Name"
	newDescription := "Updated description"
	workflow.Description = &newDescription

	err = repo.Update(ctx, workflow)
	require.NoError(t, err)

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
	repo := postgres.NewWorkflowRepository(suite.Pool)

	// Create workflow
	fixtures := testutil.NewFixtureBuilder()
	workflow := fixtures.Workflow()
	err := repo.Create(ctx, workflow)
	require.NoError(t, err)

	// Delete workflow
	err = repo.Delete(ctx, workflow.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, workflow.ID)
	assert.Error(t, err)
}
