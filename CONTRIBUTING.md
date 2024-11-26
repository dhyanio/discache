## Contributing to DisCache

Welcome to the DisCache project! We’re excited that you’re interested in contributing. Your contributions can make a huge difference to this project, whether by improving the code, updating documentation, reporting bugs, or suggesting features. Please follow the guidelines below to ensure a smooth contribution process.

### Table of Contents
- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Guidelines](#development-guidelines)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)

### Code of Conduct
By participating in this project, you agree to abide by our Code of Conduct. Please be respectful and considerate of others.

### Getting Started
1. **Fork the repository:** Click the "Fork" button on the top-right of the repository page.
2. **Clone your fork:**
    ```bash
    git clone https://github.com/your-username/discache.git
    cd discache
    ```
3. **Install dependencies:** Ensure you have Go installed. Use the following command to install project dependencies:
    ```bash
    go mod tidy
    ```
4. **Run tests:** Run the test suite to confirm everything is set up correctly:
    ```bash
    go test ./...
    ```

### How to Contribute
1. **Feature Requests and Suggestions**
    - Open a new issue to propose new features or improvements.
    - Provide as much detail as possible, including potential use cases.
2. **Reporting Bugs**
    - Check the issues page to ensure the bug hasn’t been reported yet.
    - Open a bug report and include:
        - Steps to reproduce
        - Expected behavior
        - Actual behavior
        - Relevant logs or stack traces
3. **Submitting Code**
    - Look for open issues labeled with `help wanted` or `good first issue`.
    - Discuss your approach by commenting on the issue before starting work.
    - Follow the Development Guidelines.

### Development Guidelines
**Code Style:**
- Write clean, idiomatic Go code.
- Use `gofmt` to format your code before submitting changes.

**Testing:**
- Add unit tests for any new functionality.
- Ensure all tests pass before submitting a PR.

**Documentation:**
- Update the relevant sections in the `README.md` or create new documentation as necessary.
- Add comments to your code where it improves clarity.

**Commit Messages:**
- Use clear and descriptive commit messages.
- Follow this format:
    ```
    [Type] Short description

    Detailed explanation (if necessary).
    ```
- Examples of types:
    - `feat`: A new feature
    - `fix`: A bug fix
    - `docs`: Documentation updates
    - `test`: Adding or updating tests

### Pull Request Process
1. **Create a branch for your work:**
    ```bash
    git checkout -b feature/your-feature-name
    ```
2. **Make changes and commit your work:**
    ```bash
    git add .
    git commit -m "[Type] Add your commit message here"
    ```
3. **Push your branch to GitHub:**
    ```bash
    git push origin feature/your-feature-name
    ```
4. **Open a pull request on the main branch:**
    - Provide a clear description of the changes.
    - Reference any related issues.
5. **Wait for feedback or approval:**
    - Be responsive to comments or suggestions during the review process.

### Reporting Issues
When reporting an issue:
- Include a clear title and detailed description.
- Add relevant logs, screenshots, or steps to reproduce the problem.
- Indicate the version of DisCache you are using.

Thank you for contributing to Discache! Together, we can make this project robust and useful for everyone. ❤️
