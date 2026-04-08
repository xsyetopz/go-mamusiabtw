import {
	Badge,
	Button,
	Card,
	Code,
	Group,
	SimpleGrid,
	Stack,
	Table,
	Text,
} from "@mantine/core";
import { IconPlus, IconRefresh } from "@tabler/icons-react";
import { CopyIconButton } from "../components/CopyIconButton";
import { PageHeader } from "../components/PageHeader";
import { useDeveloperDetails } from "../developerDetails";
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
	const { enabled: devDetailsEnabled } = useDeveloperDetails();

	return (
		<Stack gap="md">
			<PageHeader
				title="Plugins"
				subtitle="Install state, trust state, and actions."
				primaryAction={
					<Button
						variant="default"
						leftSection={<IconPlus size={16} />}
						onClick={onCreatePlugin}
					>
						Create
					</Button>
				}
				secondaryActions={[
					{
						key: "reload",
						label: "Reload",
						icon: <IconRefresh size={16} />,
						onClick: onReload,
						loading: busy === "plugins:reload",
					},
				]}
			/>

			<Card className="panel-card" withBorder={true} visibleFrom="sm">
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
										{devDetailsEnabled ? (
											<Group gap="xs">
												<Code>{plugin.id}</Code>
												<CopyIconButton
													value={plugin.id}
													label="Copy plugin ID"
												/>
											</Group>
										) : null}
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

			<SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md" hiddenFrom="sm">
				{plugins.map((plugin) => (
					<Card key={plugin.id} className="panel-card" withBorder={true}>
						<Stack gap="sm">
							<Group justify="space-between" align="flex-start">
								<Stack gap={2}>
									<Text fw={700}>{plugin.name || plugin.id}</Text>
									<Text size="xs" c="dimmed">
										{plugin.version || "No version"} · {plugin.commands.length}{" "}
										commands
									</Text>
									{devDetailsEnabled ? (
										<Group gap="xs">
											<Code>{plugin.id}</Code>
											<CopyIconButton
												value={plugin.id}
												label="Copy plugin ID"
											/>
										</Group>
									) : null}
								</Stack>
								<Badge color={badgeColor(plugin.loaded)}>
									{plugin.loaded ? "Loaded" : "Not loaded"}
								</Badge>
							</Group>
							<Group gap="xs">
								<Badge
									variant="light"
									color={badgeColor(plugin.has_signature_file)}
								>
									{plugin.has_signature_file ? "Signature file" : "No file"}
								</Badge>
								<Badge variant="light" color={badgeColor(plugin.signed)}>
									{plugin.signed ? "Trusted" : "Unsigned"}
								</Badge>
							</Group>
							<Button
								variant="light"
								disabled={!signingConfigured}
								loading={busy === `plugin:sign:${plugin.id}`}
								onClick={() => onSignPlugin(plugin.id)}
							>
								Sign plugin
							</Button>
						</Stack>
					</Card>
				))}
			</SimpleGrid>
		</Stack>
	);
}
