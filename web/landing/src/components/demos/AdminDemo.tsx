import { motion } from "motion/react";
import { adminTasks } from "@/data/admin";

export function AdminDemo() {
  return (
    <div className="flex h-full w-full flex-col p-4">
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm font-medium text-foreground">
          Pixoma · 管理后台
        </span>
        <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
          2 节点
        </span>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25 }}
        className="flex flex-1 flex-col overflow-hidden rounded-lg border border-border"
      >
        <div className="grid grid-cols-3 gap-2 border-b border-border bg-muted px-3 py-2 text-xs text-muted-foreground">
          <span>任务</span>
          <span>节点</span>
          <span>状态</span>
        </div>
        {adminTasks.map((task) => (
          <div
            key={task.id}
            className="grid grid-cols-3 gap-2 border-b border-border px-3 py-2.5 text-sm last:border-0"
          >
            <span className="text-foreground">{task.title}</span>
            <span className="text-muted-foreground">{task.node}</span>
            <span className="text-muted-foreground">{task.status}</span>
          </div>
        ))}
      </motion.div>
    </div>
  );
}
