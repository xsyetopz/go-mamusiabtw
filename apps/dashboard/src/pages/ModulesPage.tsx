import { Badge, Button, Card, Group, Stack, Table, Text } from "@mantine/core";
import { IconRefresh } from "@tabler/icons-react";
import { PageHeader } from "../components/PageHeader";
import { badgeColor, kindLabel } from "../format";
import type { ModuleInfo } from "../types";

type Props = {
	modules: ModuleInfo[];
	busy: string | null;
	onReload: () => void;
	onEnable: (moduleID: string) => void;
	onDisable: (moduleID: string) => void;
	onReset: (moduleID: string) => void;
};

export function ModulesPage({
	modules,
	busy,
	onReload,
	onEnable,
	onDisable,
	onReset,
}: Props) {
	return (
		<Stack gap="lg">
			<PageHeader
				title="Modules"
				subtitle="Enable, disable, reset, or reload modules."
				action={
					<Button
						leftSection={<IconRefresh size={16} />}
						loading={busy === "modules:reload"}
						onClick={onReload}
					>
						Reload modules
					</Button>
				}
			/>
			<Card className="panel-card" withBorder={true}>
				<Table className="compact-table" striped={true} highlightOnHover={true}>
					<Table.Thead>
						<Table.Tr>
							<Table.Th>Name</Table.Th>
							<Table.Th>Kind</Table.Th>
							<Table.Th>Runtime</Table.Th>
							<Table.Th>Commands</Table.Th>
							<Table.Th>Status</Table.Th>
							<Table.Th>Actions</Table.Th>
						</Table.Tr>
					</Table.Thead>
					<Table.Tbody>
						{modules.map((module) => (
							<Table.Tr key={module.id}>
								<Table.Td>
									<Stack gap={1}>
										<Text fw={600}>{module.name || module.id}</Text>
										<Text size="xs" c="dimmed">
											{module.id}
										</Text>
									</Stack>
								</Table.Td>
								<Table.Td>{kindLabel(module.kind)}</Table.Td>
								<Table.Td>{module.runtime}</Table.Td>
								<Table.Td>{module.commands.join(", ") || "—"}</Table.Td>
								<Table.Td>
									<Badge color={badgeColor(module.enabled)}>
										{module.enabled ? "Enabled" : "Disabled"}
									</Badge>
								</Table.Td>
								<Table.Td>
									<Group gap="xs">
										<Button
											size="xs"
											variant="light"
											disabled={!module.toggleable || module.enabled}
											loading={busy === `module:enable:${module.id}`}
											onClick={() => onEnable(module.id)}
										>
											Enable
										</Button>
										<Button
											size="xs"
											variant="light"
											color="gray"
											disabled={!(module.toggleable && module.enabled)}
											loading={busy === `module:disable:${module.id}`}
											onClick={() => onDisable(module.id)}
										>
											Disable
										</Button>
										<Button
											size="xs"
											variant="subtle"
											loading={busy === `module:reset:${module.id}`}
											onClick={() => onReset(module.id)}
										>
											Reset
										</Button>
									</Group>
								</Table.Td>
							</Table.Tr>
						))}
					</Table.Tbody>
				</Table>
			</Card>
		</Stack>
	);
}
