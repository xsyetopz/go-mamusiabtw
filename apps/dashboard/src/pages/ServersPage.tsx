import {
	Avatar,
	Button,
	Card,
	Code,
	Group,
	Loader,
	SimpleGrid,
	Stack,
	Text,
} from "@mantine/core";
import {
	IconArrowRight,
	IconCrown,
	IconExternalLink,
	IconRefresh,
	IconUser,
} from "@tabler/icons-react";
import { CopyIconButton } from "../components/CopyIconButton";
import { PageHeader } from "../components/PageHeader";
import { BoolStatusIconBadge } from "../components/StatusIconBadge";
import { useDeveloperDetails } from "../developerDetails";
import type { AuthMe, GuildSummary } from "../types";

type Props = {
	me: AuthMe | null;
	guilds: GuildSummary[];
	loading: boolean;
	onLogin: () => void;
	onRefresh: () => void;
	onInviteBot: () => void;
	onOpenGuild: (guildID: string) => void;
};

export function ServersPage({
	me,
	guilds,
	loading,
	onLogin,
	onRefresh,
	onInviteBot,
	onOpenGuild,
}: Props) {
	const { enabled: devDetailsEnabled } = useDeveloperDetails();

	if (!me) {
		return (
			<Stack gap="lg">
				<PageHeader
					title="Servers"
					subtitle="Sign in to see the Discord servers you can manage."
				/>
				<Card className="panel-card" withBorder={true}>
					<Stack gap="sm">
						<Text fw={700}>Discord sign-in required</Text>
						<Text size="sm" c="dimmed">
							Use your Discord account to list the servers you manage and open
							the matching dashboard.
						</Text>
						<Group>
							<Button onClick={onLogin}>Sign in with Discord</Button>
						</Group>
					</Stack>
				</Card>
			</Stack>
		);
	}

	return (
		<Stack gap="md">
			<PageHeader
				title="Servers"
				subtitle="Choose a server to check install state and setup status."
				primaryAction={
					<Button
						rightSection={<IconExternalLink size={16} />}
						onClick={onInviteBot}
					>
						Invite bot
					</Button>
				}
				secondaryActions={[
					{
						key: "refresh",
						label: "Refresh",
						icon: <IconRefresh size={16} />,
						onClick: onRefresh,
						loading,
					},
				]}
			/>
			{loading ? (
				<Group justify="center" py="xl">
					<Loader color="goblue" />
				</Group>
			) : null}
			<SimpleGrid cols={{ base: 1, md: 2, xl: 3 }} spacing="md">
				{guilds.map((guild) => (
					<Card
						key={guild.id}
						className="panel-card server-card"
						withBorder={true}
					>
						<Stack gap="md">
							<Group justify="space-between" align="flex-start">
								<Group gap="sm" align="flex-start">
									<Avatar
										src={guild.icon_url ?? null}
										radius="md"
										color="goblue"
										name={guild.name}
									/>
									<Stack gap={2}>
										<Text fw={700}>{guild.name}</Text>
										{devDetailsEnabled ? (
											<Group gap="xs">
												<Text size="xs" c="dimmed">
													ID
												</Text>
												<Code>{guild.id}</Code>
												<CopyIconButton
													value={guild.id}
													label="Copy server ID"
												/>
											</Group>
										) : null}
									</Stack>
								</Group>
								<BoolStatusIconBadge
									value={guild.bot_installed}
									labelTrue="Installed"
									labelFalse="Not installed"
								/>
							</Group>
							<Group gap="xs">
								<BoolStatusIconBadge
									value={guild.owner}
									labelTrue="Owner"
									labelFalse="Manager"
									colorTrue="goblue"
									colorFalse="gray"
									variant="light"
									iconTrue={({ size }) => <IconCrown size={size} />}
									iconFalse={({ size }) => <IconUser size={size} />}
								/>
							</Group>
							<Group justify="space-between" align="center">
								<Text size="sm" c="dimmed">
									Open the server dashboard.
								</Text>
								<Button
									rightSection={<IconArrowRight size={16} />}
									onClick={() => onOpenGuild(guild.id)}
								>
									Open
								</Button>
							</Group>
						</Stack>
					</Card>
				))}
			</SimpleGrid>
			{!loading && guilds.length === 0 ? (
				<Card className="panel-card" withBorder={true}>
					<Text size="sm" c="dimmed">
						No manageable servers were returned by Discord for this account.
					</Text>
				</Card>
			) : null}
		</Stack>
	);
}
