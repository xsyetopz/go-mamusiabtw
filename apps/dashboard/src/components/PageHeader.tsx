import {
	ActionIcon,
	Box,
	Button,
	Group,
	Menu,
	Stack,
	Text,
	Title,
} from "@mantine/core";
import { IconDots } from "@tabler/icons-react";
import type { ReactNode } from "react";

export type SecondaryAction = {
	key: string;
	label: string;
	icon?: ReactNode;
	onClick: () => void;
	disabled?: boolean;
	loading?: boolean;
};

type Props = {
	title: string;
	subtitle: string;
	action?: ReactNode;
	primaryAction?: ReactNode;
	secondaryActions?: SecondaryAction[];
	secondaryActionsLabel?: string;
};

export function PageHeader({
	title,
	subtitle,
	action,
	primaryAction,
	secondaryActions,
	secondaryActionsLabel = "More",
}: Props) {
	const secondary = secondaryActions ?? [];

	return (
		<>
			<Group justify="space-between" align="end" visibleFrom="sm">
				<Box>
					<Title order={2}>{title}</Title>
					<Text c="dimmed" size="sm">
						{subtitle}
					</Text>
				</Box>
				{(action ?? (primaryAction || secondary.length > 0)) ? (
					<Group gap="xs">
						{primaryAction}
						{secondary.map((item) => (
							<Button
								key={item.key}
								variant="default"
								size="sm"
								leftSection={item.icon}
								disabled={item.disabled ?? false}
								loading={item.loading ?? false}
								onClick={item.onClick}
							>
								{item.label}
							</Button>
						))}
					</Group>
				) : null}
			</Group>

			<Stack gap="xs" hiddenFrom="sm">
				<Box>
					<Title order={2}>{title}</Title>
					<Text c="dimmed" size="sm">
						{subtitle}
					</Text>
				</Box>
				{(action ?? (primaryAction || secondary.length > 0)) ? (
					<Group justify="space-between">
						<Group gap="xs">{action ?? primaryAction}</Group>
						{secondary.length > 0 ? (
							<Menu withinPortal={true} position="bottom-end">
								<Menu.Target>
									<ActionIcon
										variant="default"
										radius="md"
										aria-label={secondaryActionsLabel}
										title={secondaryActionsLabel}
									>
										<IconDots size={16} />
									</ActionIcon>
								</Menu.Target>
								<Menu.Dropdown>
									{secondary.map((item) => (
										<Menu.Item
											key={item.key}
											leftSection={item.icon}
											disabled={item.disabled ?? false}
											onClick={item.onClick}
										>
											{item.label}
										</Menu.Item>
									))}
								</Menu.Dropdown>
							</Menu>
						) : null}
					</Group>
				) : null}
			</Stack>
		</>
	);
}
