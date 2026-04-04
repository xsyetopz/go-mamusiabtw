import { Badge, Button, Card, Group, Stack, Table, Text } from "@mantine/core";
import { IconPlus, IconRefresh } from "@tabler/icons-react";
import { PageHeader } from "../components/PageHeader";
import { badgeColor } from "../format";
import type { PluginSummary } from "../types";

type Props = {
	plugins: PluginSummary[];
	busy: string | null;
	signingConfigured: boolean;
	onReload: () => void;
	onCreatePlugin: () => void;
	onSignPlugin: (pluginID: string) => void;
};

export function PluginsPage({
	plugins,
	busy,
	signingConfigured,
	onReload,
	onCreatePlugin,
	onSignPlugin,
}: Props) {
	return (
		<Stack gap="lg">
			<PageHeader
				title="Plugins"
				subtitle="Installed plugins, trust state, and plugin actions."
				action={
					<Group gap="xs">
						<Button
							variant="default"
							leftSection={<IconPlus size={16} />}
							onClick={onCreatePlugin}
						>
							Create plugin
						</Button>
						<Button
							leftSection={<IconRefresh size={16} />}
							loading={busy === "plugins:reload"}
							onClick={onReload}
						>
							Reload plugins
						</Button>
					</Group>
				}
			/>
			<Card className="panel-card" withBorder={true}>
				<Table className="compact-table" striped={true} highlightOnHover={true}>
					<Table.Thead>
						<Table.Tr>
							<Table.Th>Plugin</Table.Th>
							<Table.Th>Version</Table.Th>
							<Table.Th>Commands</Table.Th>
							<Table.Th>Status</Table.Th>
							<Table.Th>Signature</Table.Th>
							<Table.Th>Actions</Table.Th>
						</Table.Tr>
					</Table.Thead>
					<Table.Tbody>
						{plugins.map((plugin) => (
							<Table.Tr key={plugin.id}>
								<Table.Td>
									<Stack gap={1}>
										<Text fw={600}>{plugin.name || plugin.id}</Text>
										<Text size="xs" c="dimmed">
											{plugin.id}
										</Text>
									</Stack>
								</Table.Td>
								<Table.Td>{plugin.version || "—"}</Table.Td>
								<Table.Td>{plugin.commands.join(", ") || "—"}</Table.Td>
								<Table.Td>
									<Badge color={badgeColor(plugin.loaded)}>
										{plugin.loaded ? "Loaded" : "Not loaded"}
									</Badge>
								</Table.Td>
								<Table.Td>
									<Group gap="xs">
										<Badge color={badgeColor(plugin.has_signature_file)}>
											{plugin.has_signature_file ? "File present" : "No file"}
										</Badge>
										<Badge color={badgeColor(plugin.signed)}>
											{plugin.signed ? "Trusted" : "Unsigned"}
										</Badge>
									</Group>
								</Table.Td>
								<Table.Td>
									<Button
										size="xs"
										variant="light"
										disabled={!signingConfigured}
										loading={busy === `plugin:sign:${plugin.id}`}
										onClick={() => onSignPlugin(plugin.id)}
									>
										Sign
									</Button>
								</Table.Td>
							</Table.Tr>
						))}
					</Table.Tbody>
				</Table>
			</Card>
		</Stack>
	);
}
