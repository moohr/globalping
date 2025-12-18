import { IconButton, Tooltip } from "@mui/material";
import PauseIcon from "@mui/icons-material/Pause";
import StopIcon from "@mui/icons-material/Stop";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";

export function PlayPauseButton(props: {
  running: boolean;
  onToggle: (prevState: boolean, newState: boolean) => void;
}) {
  const { running, onToggle } = props;
  return (
    <Tooltip title={running ? "Running" : "Stopped"}>
      <IconButton
        onClick={() => {
          onToggle(running, !running);
        }}
      >
        {running ? <PauseIcon /> : <PlayArrowIcon />}
      </IconButton>
    </Tooltip>
  );
}

export function StopButton(props: {
  stopped: boolean;
  onToggle: (prevState: boolean, newState: boolean) => void;
}) {
  const { stopped, onToggle } = props;
  return (
    <Tooltip title={stopped ? "Re-Start" : "Stop"}>
      <IconButton
        onClick={() => {
          onToggle(stopped, !stopped);
        }}
      >
        {stopped ? <PlayArrowIcon /> : <StopIcon />}
      </IconButton>
    </Tooltip>
  );
}
