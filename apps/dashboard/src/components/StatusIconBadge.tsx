import { Badge, Tooltip } from "@mantine/core";
import {
	IconCircleCheck,
	IconCircleX,
} from "@tabler/icons-react";
import type { ReactNode } from "react";

type Props = {
	label: string;
	color: string;
	icon: ReactNode;
	variant?: "filled" | "light" | "outline";
	size?: "xs" | "sm" | "md" | "lg";
};

type StatusIconRendererProps = {
	size: number;
};

export function StatusIconBadge({
	label,
	color,
	icon,
	variant = "filled",
	size = "sm",
}: Props) {
	return (
		<Tooltip label={label} withArrow={true}>
			<Badge
				className="status-badge"
				color={color}
				variant={variant}
				size={size}
				radius="sm"
				aria-label={label}
				title={label}
			>
				{icon}
			</Badge>
		</Tooltip>
	);
}

type BoolProps = {
	value: boolean;
	labelTrue: string;
	labelFalse: string;
	colorTrue?: string;
	colorFalse?: string;
	iconTrue?: (props: StatusIconRendererProps) => ReactNode;
	iconFalse?: (props: StatusIconRendererProps) => ReactNode;
	variant?: "filled" | "light" | "outline";
	size?: "xs" | "sm" | "md" | "lg";
};

export function BoolStatusIconBadge({
	value,
	labelTrue,
	labelFalse,
	colorTrue = "goblue",
	colorFalse = "gray",
	iconTrue,
	iconFalse,
	variant = "filled",
	size = "sm",
}: BoolProps) {
	const label = value ? labelTrue : labelFalse;
	const color = value ? colorTrue : colorFalse;
	const Icon = value ? iconTrue : iconFalse;
	const iconNode = Icon ? (
		Icon({ size: 14 })
	) : value ? (
		<IconCircleCheck size={14} />
	) : (
		<IconCircleX size={14} />
	);
	return (
		<StatusIconBadge
			label={label}
			color={color}
			icon={iconNode}
			variant={variant}
			size={size}
		/>
	);
}
