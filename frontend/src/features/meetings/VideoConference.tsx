import React, { useState, useEffect, useRef } from 'react';
import { Video, VideoOff, Mic, MicOff, PhoneOff, Monitor, Settings } from 'lucide-react';
import { Device } from 'twilio-video';

interface VideoConferenceProps {
  roomName: string;
  token: string; // JWT token from backend
  onLeave: () => void;
}

export const VideoConference: React.FC<VideoConferenceProps> = ({ roomName, token, onLeave }) => {
  const [room, setRoom] = useState<any>(null);
  const [participants, setParticipants] = useState<any[]>([]);
  const [isVideoEnabled, setIsVideoEnabled] = useState(true);
  const [isAudioEnabled, setIsAudioEnabled] = useState(true);
  const [isScreenSharing, setIsScreenSharing] = useState(false);

  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRefs = useRef<Record<string, HTMLVideoElement>>({});

  useEffect(() => {
    connectToRoom();
    return () => {
      if (room) {
        room.disconnect();
      }
    };
  }, []);

  const connectToRoom = async () => {
    try {
      // Dynamic import to avoid bundling Twilio in all pages
      const Video = await import('twilio-video');
      
      const twilioRoom = await Video.connect(token, {
        name: roomName,
        audio: true,
        video: { width: 1280, height: 720 },
      });

      setRoom(twilioRoom);

      // Attach local participant's video
      twilioRoom.localParticipant.videoTracks.forEach((publication: any) => {
        if (localVideoRef.current && publication.track) {
          const track = publication.track;
          localVideoRef.current.appendChild(track.attach());
        }
      });

      // Handle remote participants
      twilioRoom.participants.forEach(participantConnected);

      twilioRoom.on('participantConnected', participantConnected);
      twilioRoom.on('participantDisconnected', participantDisconnected);

      // Handle disconnect
      twilioRoom.on('disconnected', () => {
        twilioRoom.localParticipant.tracks.forEach((publication: any) => {
          const track = publication.track;
          track.stop();
          const attachedElements = track.detach();
          attachedElements.forEach((element: any) => element.remove());
        });
      });

    } catch (error) {
      console.error('Failed to connect to video room:', error);
    }
  };

  const participantConnected = (participant: any) => {
    setParticipants(prev => [...prev, participant]);

    participant.tracks.forEach((publication: any) => {
      if (publication.track) {
        attachTrack(publication.track, participant.identity);
      }
    });

    participant.on('trackSubscribed', (track: any) => {
      attachTrack(track, participant.identity);
    });

    participant.on('trackUnsubscribed', (track: any) => {
      detachTrack(track);
    });
  };

  const participantDisconnected = (participant: any) => {
    setParticipants(prev => prev.filter(p => p.identity !== participant.identity));
    
    participant.tracks.forEach((publication: any) => {
      if (publication.track) {
        detachTrack(publication.track);
      }
    });
  };

  const attachTrack = (track: any, identity: string) => {
    const videoElement = remoteVideoRefs.current[identity];
    if (videoElement && track.kind === 'video') {
      videoElement.appendChild(track.attach());
    }
  };

  const detachTrack = (track: any) => {
    track.detach().forEach((element: any) => element.remove());
  };

  const toggleVideo = () => {
    if (room) {
      room.localParticipant.videoTracks.forEach((publication: any) => {
        if (publication.track) {
          if (isVideoEnabled) {
            publication.track.disable();
          } else {
            publication.track.enable();
          }
        }
      });
      setIsVideoEnabled(!isVideoEnabled);
    }
  };

  const toggleAudio = () => {
    if (room) {
      room.localParticipant.audioTracks.forEach((publication: any) => {
        if (publication.track) {
          if (isAudioEnabled) {
            publication.track.disable();
          } else {
            publication.track.enable();
          }
        }
      });
      setIsAudioEnabled(!isAudioEnabled);
    }
  };

  const shareScreen = async () => {
    if (!room) return;

    try {
      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: true,
        audio: false,
      });

      const screenTrack = stream.getTracks()[0];
      
      room.localParticipant.publishTrack(screenTrack);
      setIsScreenSharing(true);

      screenTrack.onended = () => {
        room.localParticipant.unpublishTrack(screenTrack);
        setIsScreenSharing(false);
      };
    } catch (error) {
      console.error('Failed to share screen:', error);
    }
  };

  const leaveRoom = () => {
    if (room) {
      room.disconnect();
    }
    onLeave();
  };

  return (
    <div className="h-screen bg-gray-900 flex flex-col">
      {/* Header */}
      <div className="bg-gray-800 border-b border-gray-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-white font-semibold">Meeting with Your Advisor</h2>
            <p className="text-sm text-gray-400">Room: {roomName}</p>
          </div>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            <span>{participants.length + 1} participants</span>
          </div>
        </div>
      </div>

      {/* Video Grid */}
      <div className="flex-1 p-6 overflow-hidden">
        <div className={`grid gap-4 h-full ${
          participants.length === 0 ? 'grid-cols-1' :
          participants.length === 1 ? 'grid-cols-2' :
          'grid-cols-2 grid-rows-2'
        }`}>
          {/* Local Video */}
          <div className="relative bg-gray-800 rounded-xl overflow-hidden">
            <video
              ref={localVideoRef}
              autoPlay
              playsInline
              muted
              className="w-full h-full object-cover"
            />
            <div className="absolute bottom-4 left-4 bg-black bg-opacity-60 px-3 py-1 rounded-lg">
              <span className="text-white text-sm font-medium">You</span>
            </div>
            {!isVideoEnabled && (
              <div className="absolute inset-0 bg-gray-700 flex items-center justify-center">
                <div className="text-center">
                  <div className="w-20 h-20 bg-indigo-600 rounded-full flex items-center justify-center mx-auto mb-2">
                    <span className="text-3xl text-white font-bold">You</span>
                  </div>
                  <p className="text-gray-300 text-sm">Camera Off</p>
                </div>
              </div>
            )}
          </div>

          {/* Remote Videos */}
          {participants.map((participant) => (
            <div key={participant.identity} className="relative bg-gray-800 rounded-xl overflow-hidden">
              <video
                ref={(el) => {
                  if (el) remoteVideoRefs.current[participant.identity] = el;
                }}
                autoPlay
                playsInline
                className="w-full h-full object-cover"
              />
              <div className="absolute bottom-4 left-4 bg-black bg-opacity-60 px-3 py-1 rounded-lg">
                <span className="text-white text-sm font-medium">{participant.identity}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Controls */}
      <div className="bg-gray-800 border-t border-gray-700 px-6 py-4">
        <div className="flex items-center justify-center gap-4">
          <button
            onClick={toggleAudio}
            className={`p-4 rounded-full transition-colors ${
              isAudioEnabled
                ? 'bg-gray-700 hover:bg-gray-600 text-white'
                : 'bg-red-600 hover:bg-red-700 text-white'
            }`}
            title={isAudioEnabled ? 'Mute' : 'Unmute'}
          >
            {isAudioEnabled ? <Mic className="w-6 h-6" /> : <MicOff className="w-6 h-6" />}
          </button>

          <button
            onClick={toggleVideo}
            className={`p-4 rounded-full transition-colors ${
              isVideoEnabled
                ? 'bg-gray-700 hover:bg-gray-600 text-white'
                : 'bg-red-600 hover:bg-red-700 text-white'
            }`}
            title={isVideoEnabled ? 'Stop Video' : 'Start Video'}
          >
            {isVideoEnabled ? <Video className="w-6 h-6" /> : <VideoOff className="w-6 h-6" />}
          </button>

          <button
            onClick={shareScreen}
            className={`p-4 rounded-full transition-colors ${
              isScreenSharing
                ? 'bg-indigo-600 hover:bg-indigo-700 text-white'
                : 'bg-gray-700 hover:bg-gray-600 text-white'
            }`}
            title="Share Screen"
          >
            <Monitor className="w-6 h-6" />
          </button>

          <button
            onClick={leaveRoom}
            className="p-4 rounded-full bg-red-600 hover:bg-red-700 text-white transition-colors ml-4"
            title="Leave Meeting"
          >
            <PhoneOff className="w-6 h-6" />
          </button>
        </div>
      </div>
    </div>
  );
};

// Simplified version for scheduling meetings
export const ScheduleMeetingButton: React.FC<{ advisorId: string }> = ({ advisorId }) => {
  const [isScheduling, setIsScheduling] = useState(false);

  const scheduleMeeting = async () => {
    setIsScheduling(true);
    try {
      const response = await fetch('/api/meetings/schedule', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          advisorId,
          scheduledFor: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), // Tomorrow
          duration: 30,
        }),
      });

      if (response.ok) {
        alert('Meeting scheduled successfully!');
      }
    } catch (error) {
      console.error('Failed to schedule meeting:', error);
    } finally {
      setIsScheduling(false);
    }
  };

  return (
    <button
      onClick={scheduleMeeting}
      disabled={isScheduling}
      className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
    >
      <Video className="w-5 h-5" />
      {isScheduling ? 'Scheduling...' : 'Schedule Video Meeting'}
    </button>
  );
};
