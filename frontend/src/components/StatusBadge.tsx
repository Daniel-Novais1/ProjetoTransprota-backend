import React from "react";

interface StatusBadgeProps {
  status: "Em trânsito" | "No Terminal" | "Offline";
  busId?: string;
  terminal?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({
  status,
  busId,
  terminal,
}) => {
  const getColor = () => {
    switch (status) {
      case "Em trânsito":
        return "bg-yellow-100 text-yellow-800 border-yellow-300 animate-pulse";
      case "No Terminal":
        return "bg-green-100 text-green-800 border-green-300";
      case "Offline":
        return "bg-red-100 text-red-800 border-red-300";
      default:
        return "bg-gray-100 text-gray-800 border-gray-300";
    }
  };

  return (
    <div className={`inline-flex items-center gap-2 px-3 py-1 rounded-full border ${getColor()}`}>
      <span className="text-sm font-semibold">{status}</span>
      {busId && <span className="text-xs opacity-70">#{busId}</span>}
      {terminal && status === "No Terminal" && (
        <span className="text-xs font-medium">📍 {terminal}</span>
      )}
    </div>
  );
};

export default StatusBadge;
