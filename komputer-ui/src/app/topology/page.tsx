"use client";

import { motion } from "framer-motion";
import { TopologyGraph } from "@/components/topology/topology-graph";

export default function TopologyPage() {
  return (
    <div className="flex h-full flex-col">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1, ease: "easeOut" }}
        className="flex-1"
      >
        <TopologyGraph />
      </motion.div>
    </div>
  );
}
