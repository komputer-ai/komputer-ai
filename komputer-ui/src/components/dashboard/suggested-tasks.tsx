"use client";

import { motion } from "framer-motion";
import {
  GitPullRequest,
  Bug,
  Mail,
  TestTube,
  FileCode,
  Globe,
  Activity,
  GitBranch,
  Sparkles,
  ArrowRight,
  type LucideIcon,
} from "lucide-react";
import {
  useCreateAgentModal,
  type AgentTemplate,
} from "@/lib/create-agent-modal-context";

interface SuggestedTask {
  id: string;
  title: string;
  description: string;
  icon: LucideIcon;
  template: AgentTemplate;
}

const SUGGESTED_TASKS: SuggestedTask[] = [
  {
    id: "review-pr",
    title: "Review a Pull Request",
    description:
      "Clone a repo, review a PR for bugs, security issues, and code quality.",
    icon: GitPullRequest,
    template: {
      name: "review-pr",
      instructions:
        "Review the pull request at <PASTE_PR_LINK_HERE>. Focus on:\n- Security vulnerabilities and input validation\n- Logic errors and edge cases\n- Code quality and maintainability\n\nProvide a structured summary of your findings with severity levels.",
      model: "claude-sonnet-4-6",
      lifecycle: "AutoDelete",
    },
  },
  {
    id: "debug-pods",
    title: "Debug Failing Pods",
    description:
      "Use kubectl to investigate CrashLoopBackOff pods, check events and logs.",
    icon: Bug,
    template: {
      name: "debug-pods",
      instructions:
        "Use kubectl to investigate pods that are failing in the cluster. Check for:\n- CrashLoopBackOff, Pending, or Evicted states\n- Recent events and container logs (including previous container logs)\n- Resource limits, OOMKilled signals, and scheduling issues\n\nDiagnose root causes and suggest specific fixes.",
      model: "claude-sonnet-4-6",
      lifecycle: "AutoDelete",
    },
  },
  {
    id: "scan-emails",
    title: "Scan Emails for Action Items",
    description:
      "Review recent emails, extract action items, deadlines, and follow-ups.",
    icon: Mail,
    template: {
      name: "scan-emails",
      instructions:
        "Review my recent emails and extract:\n- Action items assigned to me\n- Upcoming deadlines and due dates\n- Follow-ups I need to send\n- Meeting requests that need a response\n\nOrganize the results by priority (urgent, this week, later) and present as a checklist.",
      model: "claude-sonnet-4-6",
      lifecycle: "AutoDelete",
    },
  },
  {
    id: "test-coverage",
    title: "Analyze Test Coverage",
    description:
      "Clone a repo, run tests with coverage, identify untested critical paths.",
    icon: TestTube,
    template: {
      name: "test-coverage",
      instructions:
        "Clone the repository at <PASTE_REPO_URL_HERE> and analyze its test coverage:\n1. Install dependencies and run the test suite with coverage reporting\n2. Identify the modules and functions with the lowest coverage\n3. Flag critical code paths (error handling, auth, data mutations) that lack tests\n4. Suggest the highest-impact tests to write next\n\nProvide a summary table with coverage percentages per module.",
      model: "claude-sonnet-4-6",
      lifecycle: "AutoDelete",
    },
  },
  {
    id: "migration-script",
    title: "Write a Migration Script",
    description:
      "Write a data migration script with validation and rollback logic.",
    icon: FileCode,
    template: {
      name: "migration-script",
      instructions:
        "Write a Python migration script to transform data from <DESCRIBE_OLD_FORMAT> to <DESCRIBE_NEW_FORMAT>.\n\nRequirements:\n- Read from the source, validate each record, and write to the target format\n- Include a dry-run mode that reports what would change without applying\n- Add rollback logic to revert if something goes wrong mid-migration\n- Test with sample data and log progress\n\nSave the script and sample data to /workspace.",
      model: "claude-sonnet-4-6",
      lifecycle: "AutoDelete",
    },
  },
  {
    id: "competitive-research",
    title: "Competitive Landscape Research",
    description:
      "Spin up workers to research competitors' pricing, features, and positioning.",
    icon: Globe,
    template: {
      name: "competitive-research",
      instructions:
        "Research the competitive landscape for <DESCRIBE_YOUR_PRODUCT_OR_MARKET>.\n\nSpin up a worker agent for each of the top 5 competitors to research in parallel:\n- Product features and capabilities\n- Pricing model and tiers\n- Target audience and market positioning\n- Strengths and weaknesses\n\nOnce all workers complete, compile a comparison matrix and strategic summary with recommendations.",
      model: "claude-opus-4-6",
      lifecycle: "default",
      role: "manager",
    },
  },
  {
    id: "audit-services",
    title: "Audit Microservices Health",
    description:
      "Spin up workers per service to check health, resources, and errors.",
    icon: Activity,
    template: {
      name: "audit-services",
      instructions:
        "Audit the health of all microservices in the cluster.\n\nFor each service/deployment, spin up a worker agent to check:\n- Pod status, restart counts, and recent events\n- CPU and memory usage vs. configured limits\n- Error rates in recent logs (last 1 hour)\n- Pending or stuck rollouts\n\nCompile a health dashboard summary with a traffic-light status (green/yellow/red) per service and actionable recommendations for any issues found.",
      model: "claude-opus-4-6",
      lifecycle: "default",
      role: "manager",
    },
  },
  {
    id: "parallel-refactor",
    title: "Parallel Refactor Across Modules",
    description:
      "Identify deprecated API usage, spin up workers to migrate each module.",
    icon: GitBranch,
    template: {
      name: "parallel-refactor",
      instructions:
        "Clone the repository at <PASTE_REPO_URL_HERE> and refactor all usage of the deprecated <DESCRIBE_OLD_API> to the new <DESCRIBE_NEW_API>.\n\n1. Scan the codebase and identify all modules that use the deprecated API\n2. Spin up a worker agent for each module to perform the migration independently\n3. Each worker should update the code, run relevant tests, and report results\n4. Once all workers complete, verify overall test suite passes and create a summary of all changes made.",
      model: "claude-opus-4-6",
      lifecycle: "default",
      role: "manager",
    },
  },
];

export function SuggestedTasks() {
  const { openWithTemplate } = useCreateAgentModal();

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.2 }}
      className="relative rounded-xl overflow-hidden"
    >
      {/* Gradient border glow */}
      <div className="absolute inset-0 rounded-xl bg-gradient-to-br from-[var(--color-brand-blue)]/20 via-transparent to-[var(--color-brand-violet)]/20 pointer-events-none" />
      <div className="absolute inset-[1px] rounded-xl bg-[var(--color-bg)] pointer-events-none" />

      <div className="relative p-5">
        {/* Header */}
        <div className="flex items-center gap-2.5 mb-1">
          <motion.div
            className="flex h-7 w-7 items-center justify-center rounded-lg"
            style={{
              background: "linear-gradient(135deg, var(--color-brand-blue), var(--color-brand-violet), var(--color-brand-blue))",
              backgroundSize: "200% 200%",
            }}
            animate={{
              backgroundPosition: ["0% 0%", "100% 100%", "0% 0%"],
              boxShadow: [
                "0 0 16px rgba(63,133,217,0.4), 0 0 32px rgba(139,92,246,0.3)",
                "0 0 32px rgba(63,133,217,0.7), 0 0 64px rgba(139,92,246,0.5)",
                "0 0 16px rgba(63,133,217,0.4), 0 0 32px rgba(139,92,246,0.3)",
              ],
            }}
            transition={{
              duration: 4,
              ease: "easeInOut",
              repeat: Infinity,
            }}
          >
            <Sparkles className="size-3.5 text-white" />
          </motion.div>
          <h3 className="text-sm font-semibold text-[var(--color-text)]">
            What would you like to do?
          </h3>
        </div>
        <p className="text-xs text-[var(--color-text-secondary)] mb-4 ml-[38px]">
          Pick a task to launch your first agent — or use it as a starting point.
        </p>

        {/* Task cards */}
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2.5">
          {SUGGESTED_TASKS.map((task, i) => (
            <motion.button
              key={task.id}
              onClick={() => openWithTemplate(task.template)}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              whileHover={{ y: -3, transition: { duration: 0.15 } }}
              transition={{ duration: 0.25, delay: 0.3 + i * 0.05 }}
              className="text-left group cursor-pointer"
            >
              <div className="h-full rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-3.5 transition-all duration-200 group-hover:border-[var(--color-brand-blue)]/40 group-hover:shadow-[0_0_24px_rgba(63,133,217,0.08),0_0_48px_rgba(139,92,246,0.06)]">
                <div className="flex items-center gap-2 mb-1.5">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-[var(--color-brand-blue)]/10">
                    <task.icon className="size-3.5 text-[var(--color-brand-blue)]" />
                  </div>
                  <p className="text-[13px] font-semibold text-[var(--color-text)] group-hover:text-[var(--color-brand-blue-light)] transition-colors truncate">
                    {task.title}
                  </p>
                </div>
                {task.template.role === "manager" ? (
                  <span className="inline-block text-[9px] font-medium uppercase tracking-wider px-1.5 py-0.5 rounded-full bg-[var(--color-brand-violet)]/10 text-[var(--color-brand-violet)] mb-1">
                    Multi-agent
                  </span>
                ) : (
                  <span className="inline-block text-[9px] font-medium uppercase tracking-wider px-1.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 mb-1">
                    Single agent
                  </span>
                )}
                <p className="text-[11px] text-[var(--color-text-muted)] line-clamp-2 leading-relaxed">
                  {task.description}
                </p>
                <div className="mt-2.5 flex items-center gap-1 text-[10px] font-medium text-[var(--color-brand-blue)] opacity-0 group-hover:opacity-100 transition-opacity">
                  Launch agent <ArrowRight className="size-2.5" />
                </div>
              </div>
            </motion.button>
          ))}
        </div>
      </div>
    </motion.div>
  );
}
