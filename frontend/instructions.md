# Front-End Coding Guidelines: Support Copilot

This document defines the strict architectural rules, design tokens, and coding patterns required for front-end development within this project. Adhere to these instructions implicitly for all component generation and refactoring tasks.

---

## 1. Core Constraint: No `react.Memo`

* **Rule:** Under no circumstances should you optimize components using `React.memo` or `memo()` or `useMemo()`.
* **Rationale:** Avoid premature optimization patterns. Rely entirely on optimized state placement, structural composition, and the automatic compiler optimizations of the project's build stack.

---

## 2. Structural Rule: Separation of Concerns (Logic vs. Render)

Components must clearly decouple business logic, state lifecycle orchestration, and API interactions from the visual presentation layer. 

### Implementation Pattern: Custom Hooks
Every complex view or feature container must be split into two distinct parts:
1.  **The Logic Layer (`use[Name]State.ts` or inline hook abstract):** Handles React state, data fetching, input validation, navigation hooks, and handlers.
2.  **The Render Layer (`[Name].tsx`):** A purely presentational component that consumes state and callback functions via props or hooks. It contains nothing but JSX structural components and styling definitions.

#### Anti-Pattern (DO NOT DO)
```tsx
// Mixing inline logic, state handlers, and JSX together
export const BadComponent = () => {
  const [data, setData] = useState("");
  const handleAction = () => { /* Complex backend API call or processing logic */ };
  return <button onClick={handleAction}>{data}</button>;
};

// Hook File: useComponentState.ts
export const useComponentState = () => {
  const [data, setData] = useState("");
  const handleAction = () => { /* Complex logic here */ };
  return { data, handleAction };
};

// Render File: Component.tsx
import { useComponentState } from './useComponentState';

export const ApprovedComponent = () => {
  const { data, handleAction } = useComponentState();
  return <button onClick={handleAction}>{data}</button>;
};