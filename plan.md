# Behavioral Coaching AI Platform

## Project Vision

This platform helps users build durable habits across health, mindset, discipline, relationships, and productivity. It is not a generic chatbot. It is a behavioral feedback engine that observes actions, evaluates adherence, and adapts daily plans to improve long-term consistency.

Core goals:
- Identify behavioral weaknesses through onboarding and continuous signals
- Generate adaptive daily plans aligned to behavioral capacity
- Track execution through simple self-reporting
- Recalculate adherence and automatically adjust plan scope/difficulty
- Explain plan changes transparently to build trust

## Core Product Concept

Behavioral loop:

User Input -> Behavioral State -> Daily Plan -> Execution Tracking -> Adherence Evaluation -> Plan Adaptation -> Next Plan

AI assists reasoning, but the system is driven by a deterministic behavioral state engine that stores history and signals.

## Technical Architecture Overview

### System Structure

Go monolith with:
- REST API backend
- Embedded SvelteKit SPA frontend
- AI orchestration via Google Genkit

Data and infrastructure:
- Postgres (persistent state)
- Valkey (rate limiting / caching)

### Proposed Folder Structure

Add:

internal/
  domain/
    behavior/
      model.go
      service.go
      adherence.go
      planner.go
      adaptation.go

  ai/
    planning_agent.go
    adaptation_agent.go
    intake_agent.go

## Core Backend Domains

### Behavioral Domain (Primary Domain)

Responsibilities:
- Behavioral profile management
- Daily plan generation
- Task execution tracking
- Adherence scoring
- Adaptive plan recalculation

### AI Agent Layer

Encapsulates all LLM interactions:
- Intake agent
- Planning agent
- Adaptation agent
- Coaching explanation agent

### Infrastructure Services

- Authentication and sessions
- Rate limiting
- Background jobs / cron
- Static frontend serving

## Core Data Models

Required early for the behavioral loop:

- BehaviorProfile: psychological state, goals, constraints
- DailyPlan: AI-generated daily actions
- ExecutionLog: user-reported completion
- AdherenceState: computed performance metrics
- AdaptationLog: reasons and parameters for plan adjustments

## System Execution Flow

### Onboarding

1. User completes psychometric + conversational intake
2. AI generates initial BehaviorProfile
3. System generates first DailyPlan

### Daily Operation

1. User completes tasks and logs execution
2. System recomputes adherence metrics
3. If difficulty mismatch detected:
   - Adaptation engine modifies plan parameters
4. New plan generated automatically

### Continuous Learning

BehaviorProfile is periodically updated using:
- adherence trends
- user reflection inputs
- behavioral signals

## Behavioral Service Responsibilities

Behavior Service (central orchestrator) functions:

- CreateOrUpdateProfile()
- GenerateDailyPlan()
- LogExecution()
- RecomputeAdherence()
- AdaptUserPlan()

AdaptUserPlan() should run automatically after recompute, not only manually. Users forgetting to recompute is itself a signal, but the system must still evolve.

### On-Write Trigger Flow (Recommended)

When user checks tasks:

LogExecution()
  -> RecomputeAdherence()
    -> if mismatch detected:
         AdaptUserPlan()

Add a nightly cron reinforcement job later to prevent inactive users from freezing.

## Transparent Adaptation Mechanism

Whenever adaptation happens, store:
- adaptation_reason
- difficulty_change
- trigger_metrics

Expose a clear explanation to users, for example:
"Your plan was adjusted because your completion rate dropped below 40%."

This builds trust and improves retention.

## Strategic Insight

Most habit apps generate plans. Serious behavior platforms maintain a continuously evolving behavioral state machine.

Over time, the most important code will be:
- adherence scoring logic
- difficulty scaling math
- relapse detection heuristics

AI suggests actions; the behavior engine decides when suggestions are accepted.

## Supporting Productivity Modules (Post-Core)

These modules provide ongoing value after users stabilize their goals. They are built after the core behavioral loop is complete, and they feed signals back into the behavioral engine.

Modules:
- Journal: reflection prompts and sentiment signals
- Daily Schedule: time allocation and routine adherence
- Pomodoro: focus intervals, distraction tracking, and session consistency
- Habit Tracker: non-core micro-habits that reinforce adherence patterns
- Focus Sessions: planned deep-work blocks tied to plan objectives

Signal integration:
- Journal -> reflection inputs, mood trends
- Daily Schedule -> time adherence, overload detection
- Pomodoro -> focus endurance, session reliability
- Habit Tracker -> consistency streaks, relapse warnings
- Focus Sessions -> goal alignment, execution depth

All signals update BehaviorProfile and AdherenceState, keeping the behavioral loop the system of record.

## Development Phases (Do One at a Time)

Each phase has clear exit criteria so you can complete them sequentially.

### Phase 1 - Behavioral Domain Foundation

Scope:
- Define behavior domain models
- Implement behavior service skeleton
- Create database schema for behavioral entities
- Implement execution logging pipeline

Exit criteria:
- BehaviorProfile, DailyPlan, ExecutionLog, AdherenceState, AdaptationLog models exist
- Tables and migrations created
- Behavior service compiles with stubbed methods
- Execution logging writes to ExecutionLog

### Phase 2 - AI Integration Layer

Scope:
- Implement planning agent
- Implement adaptation agent
- Validate structured outputs

Exit criteria:
- Structured output validation in place

### Phase 3 - Behavioral Decision Loop

Scope:
- Implement adherence scoring engine
- Implement automatic plan adaptation
- Generate adaptive daily plans

Exit criteria:
- LogExecution -> RecomputeAdherence -> AdaptUserPlan chain works
- Daily plan changes when adherence thresholds crossed
- AdaptationLog persisted on change

### Phase 4 - User Interaction Layer

Scope:
- Build onboarding intake flows
- Implement daily plan UI
- Implement execution tracking UI
- Show transparent adaptation explanations

Exit criteria:
- Users can onboard and see a plan
- Users can log execution
- UI displays adaptation reasons

### Phase 5 - Automation and Reinforcement

Scope:
- Add scheduled reinforcement jobs
- Implement notification/nudge system
- Add relapse detection logic

Exit criteria:
- Scheduled job triggers plan reinforcement
- Nudge system can send reminders
- Relapse rule identifies drop-offs

### Phase 6 - Optimization and Scaling

Scope:
- Behavioral analytics dashboards
- Model tuning and prompt optimization
- Performance optimization
- Horizontal scalability preparation

Exit criteria:
- Core metrics visible in a dashboard
- Prompt changes tracked and evaluated
- Known bottlenecks addressed
- Scalability plan documented

### Phase 7 - Supporting Productivity Modules

Scope:
- Implement Journal module
- Implement Daily Schedule module
- Implement Pomodoro module
- Implement Habit Tracker module
- Implement Focus Sessions module
- Integrate signals into BehaviorProfile and AdherenceState

Exit criteria:
- Each module has UI, persistence, and basic analytics
- Module events create behavioral signals
- Behavior engine consumes signals without breaking the core loop

## Immediate Engineering Priorities

1. Implement behavioral domain module
2. Introduce AI agent abstraction layer
3. Build daily planning pipeline
4. Implement execution logging and adherence recomputation
5. Connect adaptation engine to planning engine

Completing these establishes the core behavioral loop.

Productivity modules follow after the core loop is stable.

## What To Implement Next (Exact Order)

1. Behavior domain models and tables
2. Behavior service skeleton
3. AI agent wrappers
4. GenerateDailyPlan pipeline
5. LogExecution -> RecomputeAdherence -> AdaptUserPlan chain

Once that loop works for one user, the product foundation is complete.