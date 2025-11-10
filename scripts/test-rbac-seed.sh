#!/bin/bash
set -e

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}================================${NC}"
echo -e "${YELLOW}RBAC Seeding Test Script${NC}"
echo -e "${YELLOW}================================${NC}"
echo ""

# Check if Docker is running
if ! docker ps > /dev/null 2>&1; then
    echo -e "${RED}✗ Docker is not running${NC}"
    echo "Please start Docker and try again"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"

# Start PostgreSQL if not running
echo ""
echo "Starting PostgreSQL..."
docker-compose up -d postgres
sleep 3

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PostgreSQL is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}✗ PostgreSQL failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# Run migrations
echo ""
echo "Running database migrations..."
docker-compose exec -T postgres psql -U postgres -c "CREATE DATABASE workflows;" 2>/dev/null || true
docker-compose exec -T postgres psql -U postgres -d workflows < migrations/postgres/001_initial_schema.up.sql 2>/dev/null || true
docker-compose exec -T postgres psql -U postgres -d workflows < migrations/postgres/002_auth_system.up.sql 2>/dev/null || true
echo -e "${GREEN}✓ Migrations completed${NC}"

# Test 1: Build seed command
echo ""
echo -e "${YELLOW}Test 1: Building seed command${NC}"
if go build -o bin/seed ./cmd/seed; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi

# Test 2: Show initial stats
echo ""
echo -e "${YELLOW}Test 2: Initial RBAC statistics${NC}"
if ./bin/seed --stats; then
    echo -e "${GREEN}✓ Stats command successful${NC}"
else
    echo -e "${RED}✗ Stats command failed${NC}"
    exit 1
fi

# Test 3: Verify RBAC data
echo ""
echo -e "${YELLOW}Test 3: Verifying RBAC data${NC}"
if ./bin/seed --verify; then
    echo -e "${GREEN}✓ Verification successful${NC}"
else
    echo -e "${RED}✗ Verification failed${NC}"
    exit 1
fi

# Test 4: Re-seed (test idempotency)
echo ""
echo -e "${YELLOW}Test 4: Re-seeding RBAC data (idempotency test)${NC}"
if ./bin/seed; then
    echo -e "${GREEN}✓ Re-seeding successful${NC}"
else
    echo -e "${RED}✗ Re-seeding failed${NC}"
    exit 1
fi

# Test 5: Verify after re-seeding
echo ""
echo -e "${YELLOW}Test 5: Verifying RBAC data after re-seeding${NC}"
if ./bin/seed --verify; then
    echo -e "${GREEN}✓ Verification after re-seeding successful${NC}"
else
    echo -e "${RED}✗ Verification after re-seeding failed${NC}"
    exit 1
fi

# Test 6: Check database directly
echo ""
echo -e "${YELLOW}Test 6: Direct database checks${NC}"

# Count roles
ROLE_COUNT=$(docker-compose exec -T postgres psql -U postgres -d workflows -t -c "SELECT COUNT(*) FROM roles;" | xargs)
if [ "$ROLE_COUNT" -eq 5 ]; then
    echo -e "${GREEN}✓ Correct number of roles ($ROLE_COUNT)${NC}"
else
    echo -e "${RED}✗ Incorrect number of roles (expected 5, got $ROLE_COUNT)${NC}"
    exit 1
fi

# Count permissions
PERM_COUNT=$(docker-compose exec -T postgres psql -U postgres -d workflows -t -c "SELECT COUNT(*) FROM permissions;" | xargs)
if [ "$PERM_COUNT" -eq 23 ]; then
    echo -e "${GREEN}✓ Correct number of permissions ($PERM_COUNT)${NC}"
else
    echo -e "${RED}✗ Incorrect number of permissions (expected 23, got $PERM_COUNT)${NC}"
    exit 1
fi

# Check admin has all permissions
ADMIN_PERMS=$(docker-compose exec -T postgres psql -U postgres -d workflows -t -c "SELECT COUNT(*) FROM role_permissions rp JOIN roles r ON r.id = rp.role_id WHERE r.name = 'admin';" | xargs)
if [ "$ADMIN_PERMS" -eq 23 ]; then
    echo -e "${GREEN}✓ Admin has all permissions ($ADMIN_PERMS)${NC}"
else
    echo -e "${RED}✗ Admin doesn't have all permissions (expected 23, got $ADMIN_PERMS)${NC}"
    exit 1
fi

# Test 7: Seed with users
echo ""
echo -e "${YELLOW}Test 7: Seeding with default admin user${NC}"
if ./bin/seed --users; then
    echo -e "${GREEN}✓ Seeding with users successful${NC}"
else
    echo -e "${RED}✗ Seeding with users failed${NC}"
    exit 1
fi

# Check user was created
USER_COUNT=$(docker-compose exec -T postgres psql -U postgres -d workflows -t -c "SELECT COUNT(*) FROM users WHERE username = 'admin';" | xargs)
if [ "$USER_COUNT" -eq 1 ]; then
    echo -e "${GREEN}✓ Admin user created${NC}"
else
    echo -e "${RED}✗ Admin user not created${NC}"
    exit 1
fi

# Check user has admin role
USER_ROLE=$(docker-compose exec -T postgres psql -U postgres -d workflows -t -c "SELECT COUNT(*) FROM user_roles ur JOIN users u ON u.id = ur.user_id JOIN roles r ON r.id = ur.role_id WHERE u.username = 'admin' AND r.name = 'admin';" | xargs)
if [ "$USER_ROLE" -eq 1 ]; then
    echo -e "${GREEN}✓ Admin user has admin role${NC}"
else
    echo -e "${RED}✗ Admin user doesn't have admin role${NC}"
    exit 1
fi

# Test 8: Test new permissions exist
echo ""
echo -e "${YELLOW}Test 8: Checking new permissions${NC}"

NEW_PERMS=("execution:pause" "execution:resume" "role:create" "role:read" "role:update" "role:delete" "role:assign")
for perm in "${NEW_PERMS[@]}"; do
    EXISTS=$(docker-compose exec -T postgres psql -U postgres -d workflows -t -c "SELECT EXISTS(SELECT 1 FROM permissions WHERE name = '$perm');" | xargs)
    if [ "$EXISTS" = "t" ]; then
        echo -e "${GREEN}  ✓ Permission exists: $perm${NC}"
    else
        echo -e "${RED}  ✗ Permission missing: $perm${NC}"
        exit 1
    fi
done

# Final summary
echo ""
echo -e "${YELLOW}================================${NC}"
echo -e "${GREEN}✓✓✓ All tests passed! ✓✓✓${NC}"
echo -e "${YELLOW}================================${NC}"
echo ""
echo "Summary:"
echo "  - Roles: $ROLE_COUNT"
echo "  - Permissions: $PERM_COUNT"
echo "  - Admin permissions: $ADMIN_PERMS"
echo "  - Users: $USER_COUNT"
echo ""
echo "RBAC seeding system is working correctly!"
