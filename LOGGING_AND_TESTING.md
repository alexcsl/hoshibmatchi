# Logging and Testing Implementation

This document describes the logging and testing infrastructure added to the Hoshibmatchi project.

## Logging

### Backend Services (Go)

#### Features
- **Multiple Log Levels**: DEBUG, INFO, WARN, ERROR, FATAL
- **Environment-Based Configuration**: Automatically adjusts based on `LOG_LEVEL` and `ENVIRONMENT` variables
- **Service-Specific Loggers**: Each service has its own named logger
- **Structured Logging**: Consistent format with timestamps and file information

#### Usage

**Environment Variables:**
```bash
# Set log level explicitly
LOG_LEVEL=DEBUG

# Or let it auto-detect from environment
ENVIRONMENT=development  # Uses DEBUG level
ENVIRONMENT=production   # Uses INFO level
```

**In Code:**
```go
import "github.com/hoshibmatchi/user-service/logger"

// Initialize logger
appLogger = logger.New("service-name")

// Use logger
appLogger.Debug("Debug message: %s", value)
appLogger.Info("Service started on port %d", port)
appLogger.Warn("Warning: %s", warning)
appLogger.Error("Error occurred: %v", err)
appLogger.Fatal("Fatal error: %v", err) // Exits program
```

**Log Levels:**
- `DEBUG`: Detailed information for debugging (development only)
- `INFO`: General informational messages
- `WARN`: Warning messages that don't stop execution
- `ERROR`: Error messages that need attention
- `FATAL`: Critical errors that cause program termination

#### Implementation Details

**Location:** `backend/pkg/logger/`

**Services Updated:**
- ✅ user-service
- ✅ post-service
- 📝 media-service (copy logger package)
- 📝 message-service (copy logger package)
- 📝 other services (copy logger package)

### Frontend (Vue/TypeScript)

#### Features
- **Multiple Log Levels**: DEBUG, INFO, WARN, ERROR, NONE
- **Environment-Based Configuration**: Uses `VITE_LOG_LEVEL` environment variable
- **Console Integration**: Uses native console methods
- **Development vs Production**: Different defaults for different environments

#### Usage

**Environment Variables:**
```bash
# .env.development
VITE_LOG_LEVEL=DEBUG

# .env.production
VITE_LOG_LEVEL=WARN
```

**In Code:**
```typescript
import logger from '@/utils/logger'

// Use logger
logger.debug('Debug message', data)
logger.info('User logged in', user)
logger.warn('API response slow', responseTime)
logger.error('Failed to load data', error)

// Grouping logs
logger.group('API Call')
logger.info('Request sent')
logger.debug('Headers:', headers)
logger.groupEnd()

// Performance timing
logger.time('DataLoad')
// ... some operation
logger.timeEnd('DataLoad')
```

**Creating Named Loggers:**
```typescript
import { createLogger } from '@/utils/logger'

const componentLogger = createLogger('MyComponent')
componentLogger.info('Component mounted')
```

## Unit Testing

### Backend Services (Go)

#### Test Framework
- **Standard Go testing package**
- **SQLite in-memory database** for integration tests
- **Assertions** using standard Go testing methods

#### Running Tests

```bash
# Run all tests in a service
cd backend/user-service
go test -v

# Run tests with coverage
go test -v -cover

# Run specific test
go test -v -run TestUserCreation

# Run tests for logger package
cd backend/pkg/logger
go test -v
```

#### Test Coverage

**user-service Tests:**
- ✅ User creation and validation
- ✅ Follow relationships
- ✅ Block relationships
- ✅ Email validation
- ✅ OTP generation
- ✅ Jaro-Winkler distance algorithm
- ✅ Close friend relationships

**post-service Tests:**
- ✅ Post creation
- ✅ Post likes
- ✅ Comments and replies
- ✅ Comment likes
- ✅ Collections
- ✅ Saved posts
- ✅ Hashtag regex matching
- ✅ Shared posts

**logger Tests:**
- ✅ Log level filtering
- ✅ Multiple log levels
- ✅ Logger initialization
- ✅ Log level get/set

#### Example Test
```go
func TestUserCreation(t *testing.T) {
    db, err := setupTestDB()
    if err != nil {
        t.Fatalf("Failed to setup test database: %v", err)
    }

    user := User{
        Name:     "Test User",
        Username: "testuser",
        Email:    "test@example.com",
        // ... other fields
    }

    result := db.Create(&user)
    if result.Error != nil {
        t.Fatalf("Failed to create user: %v", result.Error)
    }

    // Assertions
    if user.ID == 0 {
        t.Error("Expected user ID to be set")
    }

    if user.Username != "testuser" {
        t.Errorf("Expected username 'testuser', got '%s'", user.Username)
    }
}
```

### Frontend (Vue/TypeScript)

#### Test Framework
- **Vitest**: Fast unit test framework
- **Vue Test Utils**: Official Vue.js testing library
- **Happy DOM**: Lightweight DOM implementation

#### Running Tests

```bash
cd frontend/hoshi-vue

# Install dependencies (if not already installed)
npm install

# Run tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with UI
npm run test:ui

# Run tests with coverage
npm run test:coverage
```

#### Test Coverage

**Logger Tests:**
- ✅ Logger creation
- ✅ Log level filtering
- ✅ Multiple log levels
- ✅ Console spy verification

**Component Tests:**
- ✅ App component rendering
- ✅ Router integration

**Utility Tests:**
- ✅ Email validation
- ✅ Text truncation
- ✅ Date formatting
- ✅ API error handling

#### Example Test
```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MyComponent from '@/components/MyComponent.vue'

describe('MyComponent', () => {
  it('renders properly', () => {
    const wrapper = mount(MyComponent, {
      props: { msg: 'Hello' }
    })
    
    expect(wrapper.text()).toContain('Hello')
  })

  it('handles button click', async () => {
    const wrapper = mount(MyComponent)
    
    await wrapper.find('button').trigger('click')
    
    expect(wrapper.emitted()).toHaveProperty('click')
  })
})
```

## Configuration Files

### Backend

**Logger Package:**
- `backend/pkg/logger/logger.go` - Main logger implementation
- `backend/pkg/logger/logger_test.go` - Logger unit tests

**Service Tests:**
- `backend/user-service/main_test.go`
- `backend/post-service/main_test.go`

### Frontend

**Configuration:**
- `frontend/hoshi-vue/vitest.config.ts` - Vitest configuration
- `frontend/hoshi-vue/.env.development` - Development environment (DEBUG level)
- `frontend/hoshi-vue/.env.production` - Production environment (WARN level)

**Logger:**
- `frontend/hoshi-vue/src/utils/logger.ts` - Logger implementation
- `frontend/hoshi-vue/src/utils/__tests__/logger.test.ts` - Logger tests

**Tests:**
- `frontend/hoshi-vue/src/__tests__/App.test.ts`
- `frontend/hoshi-vue/src/__tests__/general.test.ts`

## Best Practices

### Logging

1. **Use Appropriate Log Levels**
   - DEBUG: Detailed debugging information
   - INFO: General application flow
   - WARN: Potential issues that don't stop execution
   - ERROR: Errors that need attention
   - FATAL: Critical errors (backend only)

2. **Include Context**
   ```go
   appLogger.Error("Failed to fetch user: %v, userID: %d", err, userID)
   ```

3. **Don't Log Sensitive Data**
   - Never log passwords, tokens, or personal data
   - Sanitize user input before logging

4. **Use Structured Logging**
   - Include relevant context (user ID, request ID, etc.)
   - Make logs searchable and filterable

### Testing

1. **Write Tests for Business Logic**
   - Focus on core functionality
   - Test edge cases and error conditions

2. **Use Descriptive Test Names**
   ```go
   func TestUserCreationWithValidData(t *testing.T) { ... }
   ```

3. **Keep Tests Independent**
   - Each test should set up its own data
   - Don't rely on test execution order

4. **Test Error Handling**
   - Verify error messages
   - Check error conditions

5. **Aim for Good Coverage**
   - Target 70-80% code coverage
   - Focus on critical paths

## CI/CD Integration

### Backend Tests
```bash
# Add to CI pipeline
go test ./... -v -cover
```

### Frontend Tests
```bash
# Add to CI pipeline
cd frontend/hoshi-vue
npm run test:coverage
```

## Next Steps

1. **Expand Logging to Other Services:**
   - Copy `backend/pkg/logger` to each service
   - Update imports and initialize logger in main()

2. **Add More Tests:**
   - API endpoint tests
   - Integration tests
   - E2E tests

3. **Set Up CI/CD:**
   - Configure automated testing in GitHub Actions
   - Add coverage reporting
   - Set minimum coverage thresholds

4. **Monitoring:**
   - Integrate with log aggregation service (e.g., ELK Stack)
   - Set up alerts for ERROR and FATAL logs
   - Create dashboards for log analysis

## Troubleshooting

### Backend

**Issue:** Logger not found
```bash
# Solution: Copy logger package to service
cp -r backend/pkg/logger backend/your-service/logger
```

**Issue:** Tests failing with database errors
```bash
# Solution: Ensure SQLite is available
go get gorm.io/driver/sqlite
```

### Frontend

**Issue:** Tests not running
```bash
# Solution: Install dependencies
npm install

# Ensure vitest is installed
npm install -D vitest @vitest/ui @vue/test-utils happy-dom
```

**Issue:** Logger not working
```bash
# Solution: Check environment variables
echo $VITE_LOG_LEVEL

# Restart dev server after changing .env
npm run dev
```

## Support

For questions or issues:
1. Check this documentation
2. Review test examples
3. Check console/terminal for error messages
4. Verify environment variables are set correctly
