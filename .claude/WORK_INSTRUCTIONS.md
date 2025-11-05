# How to Work with Claude on This Project

## 🎯 Quick Reference

### Starting Work on a Package

Use this exact format:

```
"Let's start Package [A/B/C/D/E/F/G/H]: [Package Name]
[Context: Dependencies complete? Starting fresh?]
Begin with [specific task]"
```

### Examples

```
"Let's start Package A: Backend Foundation
Begin with Week 1 tasks - project setup and database."
```

```
"Let's start Package C: Frontend Development
Package A is complete with API endpoints.
Begin with Vite + React + TypeScript setup."
```

```
"Let's continue Package B: Core Workflow Engine
Executor is done. Now implement the condition evaluator."
```

---

## 📦 Package Quick Reference

| Package | Name | Duration | Depends On | Can Start After |
|---------|------|----------|------------|-----------------|
| **A** | Backend Foundation | 2 weeks | None | Week 1 |
| **B** | Core Engine | 4 weeks | A | Week 3 |
| **C** | Frontend | 6-8 weeks | A | Week 3 |
| **D** | CLI Tool | 2-3 weeks | A | Week 3 |
| **E** | AI Integration | 4 weeks | A, B partial | Week 5 |
| **F** | Infrastructure | Ongoing | None | Week 1 |
| **G** | Testing | Ongoing | None | Week 1 |
| **H** | Documentation | Ongoing | A | Week 3 |

---

## 🔄 Common Commands

### Check Current Status
```
"What's the current status of all packages?"
"Show me what we've completed so far"
```

### Switch Between Packages
```
"Pause Package B. Let's switch to Package C."
"I'm done with Package A. What should we work on next?"
```

### Get Unstuck
```
"I'm blocked on [X]. What do I need to complete first?"
"What are the dependencies for Package E?"
"Can I start working on the frontend yet?"
```

### Review Progress
```
"Show me the todo list for Package B"
"What tasks are remaining in the current package?"
"Update the status of Package A to complete"
```

---

## 📋 Using TodoWrite for Tracking

I'll use TodoWrite to track tasks within each package. You can:

- See current progress at any time
- Know exactly what's in progress vs completed
- Track blocking issues

### Update Todo Status
```
"Mark the database schema task as complete"
"Add a new task: implement retry logic"
"Show me the current todo list"
```

---

## 🎨 Working Style Preferences

### If You Want Me To Plan First
```
"Let's plan Package B before implementing.
Create a detailed breakdown of tasks."
```

### If You Want Me To Just Code
```
"Let's start Package A and implement directly.
No planning needed, I'll review as we go."
```

### If You Want Step-by-Step Review
```
"Implement the workflow executor.
Show me the code before moving to the next step."
```

### If You Want Autonomous Implementation
```
"Implement all of Package A.
Check in with me when you're done or if you get blocked."
```

---

## 🔍 Context Management

### Give Me Context
If you've made changes outside our session:

```
"I've completed [task X] manually.
Package A status: database done, API in progress.
Let's continue with the repository layer."
```

### When Resuming Work
```
"We were working on Package B: Core Engine.
Last session we finished the executor.
Let's continue with the evaluator."
```

### When Starting Fresh
```
"This is a fresh start.
Package A is not started yet.
Let's begin with project setup."
```

---

## 🚨 When Things Go Wrong

### If I Make a Mistake
```
"That's not quite right. Let me clarify:
[Your clarification]
Please update the implementation."
```

### If You Change Requirements
```
"Actually, I want to change the approach for [X].
Instead of [old way], let's do [new way]."
```

### If We Need to Backtrack
```
"Let's revert the last change to [file].
We need to use [different approach] instead."
```

---

## 📂 File References

When you want me to:

### Read Specific Files
```
"Read the PARALLEL_WORK_PACKAGES.md file"
"Show me the current implementation of executor.go"
```

### Update Documentation
```
"Update PARALLEL_WORK_PACKAGES.md to mark Package A as complete"
"Add the new API endpoints to ARCHITECTURE.md"
```

### Create New Files
```
"Create internal/engine/executor.go with the workflow executor implementation"
```

---

## 🎯 Best Practices

### ✅ Do This
- Be specific about which package you want to work on
- Tell me if dependencies are complete
- Let me know if you want planning vs. implementation
- Update me on external changes
- Use the package letter/name (e.g., "Package A")

### ❌ Avoid This
- Vague requests like "let's continue" (continue with what?)
- Starting dependent packages without confirming dependencies
- Assuming I know about external changes
- Switching packages without telling me

---

## 🏃 Quick Start (Copy-Paste)

### Starting the Project From Scratch

```
"Let's start Package A: Backend Foundation.
This is a fresh start, nothing exists yet.
Begin with project setup - create the Go project structure."
```

### After Package A is Complete

```
"Package A is complete. Let's start Package C: Frontend Development.
Begin with Vite + React + TypeScript setup."
```

```
"Package A is complete. Let's start Package B: Core Workflow Engine.
Begin with the workflow executor implementation."
```

---

## 💬 Example Session Flow

```
You: "Let's start Package A: Backend Foundation. Begin with project setup."

Me: [Creates project structure, sets up Go modules, etc.]

You: "Good! Now let's create the database schema."

Me: [Creates PostgreSQL migrations and schema]

You: "Perfect. Now implement the basic CRUD API for workflows."

Me: [Implements API handlers, routes, repository layer]

You: "Package A is done! Let's start Package C: Frontend.
     Begin with Vite + React + TypeScript setup."

Me: [Switches to frontend, sets up Vite project]
```

---

## 📊 Progress Tracking

I maintain a todo list automatically. You can check it anytime:

```
"Show me the current todo list"
"What tasks are in progress?"
"What's left to complete in Package B?"
```

---

## 🎓 Pro Tips

1. **Work in packages, not weeks** - Packages are more flexible than strict weekly timelines

2. **Start multiple packages** - If you have multiple developers, tell me which package each is working on:
   ```
   "Developer 1 is starting Package A.
    Developer 2 is starting Package F (infrastructure).
    I'm Developer 1, let's do Package A."
   ```

3. **Use the reference files**:
   - `PARALLEL_WORK_PACKAGES.md` - Detailed package definitions
   - `IMPLEMENTATION_ROADMAP.md` - Weekly breakdown
   - `ARCHITECTURE.md` - Technical decisions
   - `GETTING_STARTED.md` - Setup instructions

4. **Check dependencies first**:
   ```
   "Can I start Package E yet? What are its dependencies?"
   ```

5. **Be explicit about context**:
   ```
   "We're in Week 5. Package A and B are complete.
    Let's start Package E: AI Integration."
   ```

---

## 🚀 Ready to Start?

Just tell me:
1. Which package you want to work on
2. What dependencies are complete (if any)
3. What specific task to begin with

Example:
```
"Let's start Package A: Backend Foundation.
Nothing exists yet, this is day 1.
Begin with creating the Go project structure."
```

And I'll immediately start implementing! 🎯
