# Contributing to Hoshibmatchi

Thank you for your interest in contributing to Hoshibmatchi! This document provides guidelines and instructions for contributing to this project.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

Please be respectful and constructive in all interactions. This is an educational project, and we welcome developers of all skill levels.

## Getting Started

1. **Fork the repository**
   ```bash
   git clone https://github.com/yourusername/hoshibmatchi.git
   cd hoshibmatchi
   ```

2. **Set up your development environment**
   - Follow the [README.md](README.md) for installation instructions
   - Copy `.env.example` to `.env` and configure your local environment
   - Run `docker-compose -f docker-compose.yml -f docker-compose.dev.yml up`

3. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

### Backend (Go Services)

1. Make changes to your service
2. Test locally:
   ```bash
   cd backend/your-service
   go test ./...
   go run main.go
   ```
3. Update proto files if needed:
   ```bash
   cd protos
   protoc --go_out=. --go-grpc_out=. your-service.proto
   ```

### Frontend (Vue.js)

1. Make changes to components/pages
2. Test locally:
   ```bash
   cd frontend/hoshi-vue
   npm run test
   npm run dev
   ```
3. Check for linting errors:
   ```bash
   npm run lint
   npm run lint:fix
   ```

### AI Service (Python)

1. Make changes to AI models or endpoints
2. Test locally:
   ```bash
   cd backend/ai-service
   python main.py
   ```

## Coding Standards

### Go
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Write descriptive variable and function names
- Add comments for exported functions
- Use error wrapping: `fmt.Errorf("context: %w", err)`

Example:
```go
// GetUserProfile retrieves a user's profile by username
func (s *server) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest) (*pb.GetUserProfileResponse, error) {
    if req.Username == "" {
        return nil, status.Error(codes.InvalidArgument, "Username is required")
    }
    // Implementation...
}
```

### Vue.js/TypeScript
- Use TypeScript for type safety
- Follow Vue 3 Composition API best practices
- Use `<script setup>` syntax
- Keep components focused and reusable
- Use SCSS for styling with proper scoping

Example:
```vue
<script setup lang="ts">
import { ref, computed } from "vue";

interface User {
  id: number;
  username: string;
}

const users = ref<User[]>([]);
const activeUser = computed(() => users.value[0]);
</script>
```

### Python
- Follow PEP 8 style guide
- Use type hints for function parameters
- Write docstrings for functions
- Use meaningful variable names

Example:
```python
async def summarize_caption(caption: str) -> str:
    """
    Summarizes a post caption using the AI model.
    
    Args:
        caption: The caption text to summarize
        
    Returns:
        The summarized text
    """
    # Implementation...
```

## Testing Guidelines

### Backend Tests
- Write unit tests for business logic
- Test error cases and edge cases
- Use table-driven tests in Go

```go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"Valid password", "Str0ng!Pass", false},
        {"Too short", "Short1!", true},
        {"No special char", "Str0ngPass", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("validatePassword() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Frontend Tests
- Write component tests for critical UI
- Test user interactions
- Mock API calls

```typescript
import { mount } from '@vue/test-utils';
import LoginForm from './LoginForm.vue';

describe('LoginForm', () => {
  it('validates email format', async () => {
    const wrapper = mount(LoginForm);
    await wrapper.find('input[type="email"]').setValue('invalid-email');
    await wrapper.find('form').trigger('submit');
    expect(wrapper.text()).toContain('Invalid email');
  });
});
```

## Commit Message Guidelines

Use conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Adding/updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(auth): add 2FA support

Implemented two-factor authentication using OTP codes sent via email.
Users can enable 2FA in their account settings.

Closes #123
```

```
fix(post): resolve video upload timeout

Increased max file size limit and request timeout for video uploads.

Fixes #456
```

## Pull Request Process

1. **Update your branch**
   ```bash
   git fetch origin
   git rebase origin/main
   ```

2. **Run all tests**
   ```bash
   # Backend
   cd backend/your-service && go test ./...
   
   # Frontend
   cd frontend/hoshi-vue && npm run test
   ```

3. **Create a pull request**
   - Provide a clear title and description
   - Reference any related issues
   - Include screenshots for UI changes
   - Ensure CI/CD checks pass

4. **Pull Request Template**
   ```markdown
   ## Description
   Brief description of changes

   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update

   ## How Has This Been Tested?
   Describe your testing process

   ## Screenshots (if applicable)
   Add screenshots for UI changes

   ## Checklist
   - [ ] My code follows the project's style guidelines
   - [ ] I have performed a self-review
   - [ ] I have commented my code where necessary
   - [ ] I have updated the documentation
   - [ ] My changes generate no new warnings
   - [ ] I have added tests that prove my fix/feature works
   - [ ] New and existing tests pass locally
   ```

5. **Code Review**
   - Address reviewer feedback promptly
   - Make requested changes in new commits
   - Once approved, your PR will be merged

## Areas for Contribution

We welcome contributions in these areas:

### High Priority
- [ ] Unit test coverage improvement
- [ ] API documentation enhancement
- [ ] Performance optimization
- [ ] Security hardening
- [ ] Accessibility improvements
- [ ] Architectural Refactor

### Features
- [ ] Story mentions (@username)
- [ ] Post scheduling
- [ ] Advanced search filters
- [ ] Dark/Light theme improvements
- [ ] Mobile responsive design enhancements

### Bug Fixes
- Check the [Issues](https://github.com/yourusername/hoshibmatchi/issues) page for open bugs

### Documentation
- Improve README
- Add architecture diagrams
- Create API usage examples
- Write deployment guides

## Questions?

If you have questions or need help:
- Open an issue with the `question` label
- Check existing issues and discussions
- Review the README.md and SECURITY.md

Thank you for contributing! 🎉
