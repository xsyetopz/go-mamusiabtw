import {
	Badge,
	Button,
	Card,
	Group,
	SimpleGrid,
	Stack,
	Text,
	ThemeIcon,
	Title,
} from "@mantine/core";
import {
	IconArrowRight,
	IconBrandDiscord,
	IconChecklist,
	IconServer,
	IconShieldCheck,
} from "@tabler/icons-react";
import type { AuthMe, SetupStatus } from "../types";

type Props = {
	me: AuthMe | null;
	setupStatus: SetupStatus | null;
	onLogin: () => void;
	onOpenServers: () => void;
};

export function HomePage({ me, setupStatus, onLogin, onOpenServers }: Props) {
	const loginReady = setupStatus?.login_ready ?? false;

	return (
		<Stack gap="xl">
			<section className="hero-panel">
				<Stack gap="lg">
					<Badge variant="light" color="teal" radius="sm" w="fit-content">
						Website and server dashboard
					</Badge>
					<Stack gap="xs">
						<Title order={1} className="hero-title">
							Manage the bot from the web, then jump straight into your server.
						</Title>
						<Text size="lg" c="dimmed" maw={720}>
							Sign in with Discord, pick a server you manage, add the bot, and
							check setup status without digging through config files.
						</Text>
					</Stack>
					<Group gap="sm">
						{me ? (
							<Button
								rightSection={<IconArrowRight size={16} />}
								onClick={onOpenServers}
							>
								Open your servers
							</Button>
						) : (
							<Button
								leftSection={<IconBrandDiscord size={16} />}
								onClick={onLogin}
								disabled={!loginReady}
							>
								Sign in with Discord
							</Button>
						)}
						<Button variant="default" component="a" href="#/servers">
							Server list
						</Button>
					</Group>
					{loginReady ? null : (
						<Text size="sm" c="dimmed">
							Sign-in is not ready yet. Check the setup page for the missing
							OAuth values.
						</Text>
					)}
				</Stack>
			</section>

			<SimpleGrid cols={{ base: 1, md: 3 }} spacing="md">
				<Card className="panel-card feature-card" withBorder={true}>
					<Stack gap="sm">
						<ThemeIcon variant="light" color="teal" size={40} radius="md">
							<IconServer size={20} />
						</ThemeIcon>
						<Text fw={700}>Server access</Text>
						<Text size="sm" c="dimmed">
							See only the Discord servers you can manage, then open the right
							one directly.
						</Text>
					</Stack>
				</Card>
				<Card className="panel-card feature-card" withBorder={true}>
					<Stack gap="sm">
						<ThemeIcon variant="light" color="teal" size={40} radius="md">
							<IconChecklist size={20} />
						</ThemeIcon>
						<Text fw={700}>Setup checks</Text>
						<Text size="sm" c="dimmed">
							Confirm whether the bot is installed and whether the server is
							ready for the feature set you want to use.
						</Text>
					</Stack>
				</Card>
				<Card className="panel-card feature-card" withBorder={true}>
					<Stack gap="sm">
						<ThemeIcon variant="light" color="teal" size={40} radius="md">
							<IconShieldCheck size={20} />
						</ThemeIcon>
						<Text fw={700}>Owner area</Text>
						<Text size="sm" c="dimmed">
							Bot-global controls stay separate. Owners can still use the
							internal area without mixing them into server management.
						</Text>
					</Stack>
				</Card>
			</SimpleGrid>
		</Stack>
	);
}
