# Create a plan as beads tasks

## description:

use beads tasks to plan implementation

-- 

breakdown the discussed requirements/features into individual units of work

## Agent Instructions

From the discussed requirements/features create the implementation plan using beads

The agent should:

1. **Create the epic**
```sh
bd create "[{TITLE}]" -t epic -p 1 -d "{SUMMARY}" --json
```

2. **Create the features/phases**
  - For each feature/phase
  ```sh
  bd create "[{TITLE}]" -t feature -p 2 -d "{DESCRIPTION}" --json
  ```

  - link to epic
  ```sh
  bd dep add <epic> --parent-child <feature>
  ```

3. **Link dependent features**
```sh
  bd dep add <featureB> --blocked-by <featureA>
```

4. **For each atomic task in each feature/phase**
  - each task should have the following CONTENT
    - a detailed implementation plan, including what files/code to update
    - associated tests that need updates
    - specifications / public documentation updates where appropriate

  ```sh
  bd create "[{TITLE}]" -t task -p 2 -d "{CONTENT}" --json
  ```

  - link to feature
  ```sh
  bd dep add <feature> --blocked-by <task>
  ```

